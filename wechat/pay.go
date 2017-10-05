package wechat

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	payResultSuccess = "SUCCESS"

	payCacheDuration = 2 * time.Hour
	payCacheLimit    = 1000
)

type WechatPayServer struct {
	WechatClient
	notifyMap *CacheMap
}

func NewPayServer() *WechatPayServer {
	srv := new(WechatPayServer)
	srv.notifyMap = NewCacheMap(payCacheDuration, payCacheLimit)
	return srv
}

func (srv *WechatPayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)

	// pay result callback
	if strings.HasSuffix(r.URL.Path, "/pay") &&
		r.Method == http.MethodPost {

		success, err := srv.notifyResult(r)
		if err != nil {
			log.Println(err.Error())
			return
		}
		w.Write(success)
		return
	}

	// make order
	r.ParseForm()
	p := srv.parseParam(r)
	p.Sign = srv.paySignature(p, p.Mch_key, "mch_key", "call_url")

	order, err := srv.sendParam(p)
	if err != nil {
		w.Write(JsonResponse(err))
		return
	}
	if order.Result_code != payResultSuccess ||
		order.Return_code != payResultSuccess {
		w.Write(JsonResponse(order))
		return
	}

	// store param
	srv.notifyMap.Set(p.Key(), *p)
	defer srv.notifyMap.Shrink()
	log.Printf("set key: %s\n", p.Key())
	log.Printf("call_url: %s\n", p.Call_url)

	if strings.HasSuffix(r.URL.Path, "/pay") {
		w.Write(JsonResponse(order))
		return
	}
	if strings.HasSuffix(r.URL.Path, "/qrcode") {
		_url := fmt.Sprintf("%s/qrcode?path=%s", srv.HostUrl(r), url.QueryEscape(order.Code_url))
		bs, err := HttpGetJson(_url, nil)
		if err != nil {
			w.Write(JsonResponse(err))
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "max-age=7200")
		w.Write(bs)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/js") {
		config := srv.jsConfig(p.AppId, order.Prepay_id, p.Mch_key)
		var_name := r.Form.Get("var")
		if var_name == "" {
			w.Write(JsonResponse(config))
		} else {
			bs := JsonResponse(config)
			js := fmt.Sprintf("var %s=%s", var_name, string(bs))
			w.Write([]byte(js))
		}
		return
	}
}

func (srv *WechatPayServer) parseParam(r *http.Request) (p *wxPayParam) {
	f := r.Form

	p = &wxPayParam{
		// register parameters
		AppId:            f.Get("appid"),
		Mch_id:           f.Get("mch_id"),
		Mch_key:          f.Get("mch_key"),
		Spbill_create_ip: srv.choice(f.Get("spbill_create_ip"), f.Get("server_ip")),

		// necessary parameters
		Total_fee:  srv.choice(f.Get("total_fee"), f.Get("fee")),
		Body:       srv.choice(f.Get("body"), f.Get("name"), "微信支付"),
		Notify_url: srv.NormalizeUrl(r, "/pay", ""),
		Call_url:   srv.choice(f.Get("notify_url"), f.Get("call")),

		// internal parameters
		Nonce_str:    srv.choice(f.Get("nonce_str"), randomString(32)),
		Trade_type:   srv.choice(f.Get("trade_type"), "NATIVE"),
		Out_trade_no: f.Get("out_trade_no"),

		// optional parameters
		Device_info: f.Get("device_info"),
		Detail:      CDATA(f.Get("detail")),
		Attach:      f.Get("attach"),
		Fee_type:    f.Get("fee_type"),
		Time_start:  f.Get("time_start"),
		Time_expire: f.Get("time_expire"),
		Goods_tag:   f.Get("goods_tag"),
		Product_id:  srv.choice(f.Get("product_id"), "pay"),
		Limit_pay:   f.Get("limit_pay"),
		Openid:      f.Get("openid"),
		Scene_info:  CDATA(f.Get("scene_info")),
	}

	if p.Out_trade_no == "" {
		oid := "----"
		if len(p.Openid) > 4 {
			oid = p.Openid[len(p.Openid)-4:]
		}
		p.Out_trade_no = fmt.Sprintf("%s-%s-%s",
			time.Now().Format("20060102150405"),
			oid,
			randomString(8))
	}
	if p.Call_url != "" {
		p.Call_url = srv.NormalizeUrl(r, p.Call_url, "")
	}
	if strings.HasSuffix(r.URL.Path, "/js") {
		p.Trade_type = "JSAPI"
		if p.Openid == "" {
			cookie, err := r.Cookie("openid")
			if err == nil {
				p.Openid = cookie.Value
			}
		}
	}
	return
}

func (*WechatPayServer) choice(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// doc: https://pay.weixin.qq.com/wiki/doc/api/native.php?chapter=4_3
func (srv *WechatPayServer) paySignature(p interface{}, key string, excluded ...string) string {
	t := reflect.TypeOf(p)
	v := reflect.ValueOf(p)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	is_excluded := func(s string) bool {
		for _, x := range excluded {
			if strings.ToLower(s) == strings.ToLower(x) {
				return true
			}
		}
		return false
	}

	var ss []string
	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).Kind() != reflect.String {
			continue
		}
		name := t.Field(i).Name
		name = strings.ToLower(name)
		value := v.Field(i).String()
		if value == "" {
			continue
		}
		if is_excluded(name) {
			continue
		}
		str := fmt.Sprintf("%s=%s", name, value)
		ss = append(ss, str)
	}

	sort.Strings(ss)
	sign_str := strings.Join(ss, "&") + "&key=" + key
	sign_bytes := md5.Sum([]byte(sign_str))
	return fmt.Sprintf("%X", sign_bytes[:])
}

func (srv *WechatPayServer) sendParam(p *wxPayParam) (r *wxPayOrder, err error) {
	bs, err := xml.Marshal(p)
	if err != nil {
		return
	}

	_url := "https://api.mch.weixin.qq.com/pay/unifiedorder"
	resp, err := http.Post(_url, "", bytes.NewReader(bs))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	resp_bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	r = new(wxPayOrder)
	err = xml.Unmarshal(resp_bytes, r)
	if err != nil {
		return
	}
	return
}

func (srv *WechatPayServer) notifyResult(r *http.Request) (data []byte, err error) {

	type wxPaySuccess struct {
		Return_code string `xml:"return_code"`
		Return_msg  string `xml:"return_msg"`
	}
	success := &wxPaySuccess{
		Return_code: "SUCCESS",
		Return_msg:  "OK",
	}
	data, err = xml.Marshal(success)

	// parse result
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	result := &wxPayResult{}
	err = xml.Unmarshal(body, result)
	if err != nil {
		return
	}

	// load param
	v, ok := srv.notifyMap.Get(result.Key())
	if !ok {
		log.Println(string(body))
		log.Printf("get key: %s\n", result.Key())
		log.Println(ErrCacheTimeout.Error())
		return
	}
	p := v.(wxPayParam)
	if p.Call_url == "" {
		return
	}

	// check result sign
	//...

	// notify call_url
	js, err := json.Marshal(result)
	if err != nil {
		return
	}
	resp, err := http.Post(p.Call_url, "", bytes.NewReader(js))
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status)
		return
	}

	return
}

func (srv *WechatPayServer) jsConfig(appid, prepay_id, mch_key string) interface{} {
	type wxPayJs struct {
		AppId     string `json:"appId"`
		Timestamp string `json:"timeStamp"`
		NonceStr  string `json:"nonceStr"`
		Package   string `json:"package"`
		SignType  string `json:"signType"`
		PaySign   string `json:"paySign"`
	}

	c := &wxPayJs{
		AppId:     appid,
		Timestamp: fmt.Sprintf("%d", time.Now().Unix()),
		NonceStr:  randomString(16),
		Package:   fmt.Sprintf("prepay_id=%s", prepay_id),
		SignType:  "MD5",
	}
	sign_str := fmt.Sprintf("appId=%s&nonceStr=%s&package=%s&signType=%s&timeStamp=%s&key=%s",
		c.AppId, c.NonceStr, c.Package, c.SignType, c.Timestamp, mch_key)
	sign_bytes := md5.Sum([]byte(sign_str))
	c.PaySign = fmt.Sprintf("%X", sign_bytes[:])
	return c
}

type wxPayParam struct {
	XMLName          xml.Name `xml:"xml"`
	AppId            string   `xml:"appid"`                 // *公众账号ID
	Mch_id           string   `xml:"mch_id"`                // *商户号
	Mch_key          string   `xml:"-"`                     // 商户秘钥
	Device_info      string   `xml:"device_info,omitempty"` // 设备号
	Nonce_str        string   `xml:"nonce_str"`             // *随机字符串
	Sign             string   `xml:"sign"`                  // *签名
	Sign_type        string   `xml:"sign_type,omitempty"`   // 签名类型(MD5,HMAC-SHA256)
	Body             string   `xml:"body"`                  // *商品描述
	Detail           CDATA    `xml:"detail,omitempty"`      // 商品详情
	Attach           string   `xml:"attach,omitempty"`      // 附加数据
	Out_trade_no     string   `xml:"out_trade_no"`          // *商户订单号
	Fee_type         string   `xml:"fee_type,omitempty"`    // 标价币种(CNY)
	Total_fee        string   `xml:"total_fee"`             // *标价金额
	Spbill_create_ip string   `xml:"spbill_create_ip"`      // *终端IP(APP和网页支付提交用户端ip，Native支付填调用微信支付API的机器IP。)
	Time_start       string   `xml:"time_start,omitempty"`  // 交易起始时间
	Time_expire      string   `xml:"time_expire,omitempty"` // 交易结束时间
	Goods_tag        string   `xml:"goods_tag,omitempty"`   // 订单优惠标记
	Notify_url       string   `xml:"notify_url"`            // *通知地址
	Call_url         string   `xml:"-"`                     // 用户回调地址
	Trade_type       string   `xml:"trade_type"`            // *交易类型(JSAPI,NATIVE,APP)
	Product_id       string   `xml:"product_id,omitempty"`  // 商品ID(trade_type=NATIVE时（即扫码支付），此参数必传。)
	Limit_pay        string   `xml:"limit_pay,omitempty"`   // 指定支付方式(no_credit--可限制用户不能使用信用卡支付)
	Openid           string   `xml:"openid,omitempty"`      // 用户标识(trade_type=JSAPI时（即公众号支付），此参数必传)
	Scene_info       CDATA    `xml:"scene_info,omitempty"`  // 场景信息
}

type wxPayOrder struct {
	Return_code  string `xml:"return_code" json:"return_code"`             // 返回状态码(SUCCESS/FAIL)
	Return_msg   string `xml:"return_msg" json:"return_msg"`               // 返回信息: 如非空，为错误原因
	Appid        string `xml:"appid" json:"appid"`                         // 公众账号ID
	Mch_id       string `xml:"mch_id" json:"mch_id"`                       // 商户号
	Device_info  string `xml:"device_info" json:"device_info,omitempty"`   // 设备号
	Nonce_str    string `xml:"nonce_str" json:"nonce_str"`                 // 随机字符串
	Sign         string `xml:"sign" json:"sign"`                           // 签名
	Result_code  string `xml:"result_code" json:"result_code"`             // 业务结果(SUCCESS/FAIL)
	Err_code     string `xml:"err_code" json:"err_code,omitempty"`         // 错误代码
	Err_code_des string `xml:"err_code_des" json:"err_code_des,omitempty"` // 错误代码描述
	Trade_type   string `xml:"trade_type" json:"trade_type"`               // 交易类型
	Prepay_id    string `xml:"prepay_id" json:"prepay_id"`                 // 预支付交易会话标识
	Code_url     string `xml:"code_url" json:"code_url,omitempty"`         // 二维码链接
}

type wxPayResult struct {
	Return_code          string `xml:"return_code,omitempty" json:"return_code"`                   // *返回状态码(SUCCESS/FAIL)
	Return_msg           string `xml:"return_msg,omitempty" json:"return_msg,omitempty"`           // 返回信息: 如非空，为错误原因
	Appid                string `xml:"appid" json:"appid"`                                         // *公众账号ID
	Mch_id               string `xml:"mch_id" json:"mch_id"`                                       // *商户号
	Device_info          string `xml:"device_info" json:"device_info,omitempty"`                   // 设备号
	Nonce_str            string `xml:"nonce_str" json:"nonce_str"`                                 // *随机字符串
	Sign                 string `xml:"sign" json:"sign"`                                           // *签名
	Sign_type            string `xml:"sign_type,omitempty" json:"sign_type,omitempty"`             // 签名类型(MD5,HMAC-SHA256)
	Result_code          string `xml:"result_code" json:"result_code"`                             // *业务结果(SUCCESS/FAIL)
	Err_code             string `xml:"err_code" json:"err_code,omitempty"`                         // 错误代码
	Err_code_des         string `xml:"err_code_des" json:"err_code_des,omitempty"`                 // 错误代码描述
	Openid               string `xml:"openid" json:"openid"`                                       // *用户标识
	Is_subscribe         string `xml:"is_subscribe" json:"is_subscribe,omitempty"`                 //用户是否关注公众账号(Y-关注,N-未关注)，仅在公众账号类型支付有效
	Trade_type           string `xml:"trade_type" json:"trade_type"`                               // *交易类型(JSAPI,NATIVE,APP)
	Bank_type            string `xml:"bank_type" json:"bank_type"`                                 // *付款银行
	Total_fee            string `xml:"total_fee" json:"total_fee"`                                 // *订单金额，单位为分
	Settlement_total_fee string `xml:"settlement_total_fee" json:"settlement_total_fee,omitempty"` // 应结订单金额
	Fee_type             string `xml:"fee_type" json:"fee_type,omitempty"`                         // 货币种类
	Cash_fee             string `xml:"cash_fee" json:"cash_fee"`                                   // *现金支付金额
	Cash_fee_type        string `xml:"cash_fee_type" json:"cash_fee_type,omitempty"`               // 现金支付货币类型(CNY)
	Transaction_id       string `xml:"transaction_id" json:"transaction_id"`                       // *微信支付订单号
	Out_trade_no         string `xml:"out_trade_no" json:"out_trade_no"`                           // *商户订单号
	Attach               string `xml:"attach" json:"attach,omitempty"`                             // 商家数据包
	Time_end             string `xml:"time_end" json:"time_end"`                                   // *支付完成时间

	Coupon_fee    string `xml:"coupon_fee" json:"coupon_fee,omitempty"`       // 总代金券金额
	Coupon_count  string `xml:"coupon_count" json:"coupon_count,omitempty"`   //代金券使用数量
	Coupon_type_0 string `xml:"coupon_type_0" json:"coupon_type_0,omitempty"` //代金券类型
	Coupon_id_0   string `xml:"coupon_id_0" json:"coupon_id_0,omitempty"`     //代金券ID	coupon_id_$n
	Coupon_fee_0  string `xml:"coupon_fee_0" json:"coupon_fee_0,omitempty"`   //单个代金券支付金额
	Coupon_type_1 string `xml:"coupon_type_1" json:"coupon_type_1,omitempty"` //代金券类型
	Coupon_id_1   string `xml:"coupon_id_1" json:"coupon_id_1,omitempty"`     //代金券ID	coupon_id_$n
	Coupon_fee_1  string `xml:"coupon_fee_1" json:"coupon_fee_1,omitempty"`   //单个代金券支付金额
	Coupon_type_2 string `xml:"coupon_type_2" json:"coupon_type_2,omitempty"` //代金券类型
	Coupon_id_2   string `xml:"coupon_id_2" json:"coupon_id_2,omitempty"`     //代金券ID	coupon_id_$n
	Coupon_fee_2  string `xml:"coupon_fee_2" json:"coupon_fee_2,omitempty"`   //单个代金券支付金额
}

func (p *wxPayParam) Key() string {
	return p.Mch_id + p.Out_trade_no
}

func (r *wxPayResult) Key() string {
	return r.Mch_id + r.Out_trade_no
}
