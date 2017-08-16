package wxproxy

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

const (
	payResultSuccess = "SUCCESS"
)

type wxPayParam struct {
	XMLName          xml.Name `xml:"xml"`
	AppId            string   `xml:"appid"`
	Mch_id           string   `xml:"mch_id"`
	Mch_key          string   `xml:"-"`
	Device_info      string   `xml:"device_info,omitempty"`
	Nonce_str        string   `xml:"nonce_str"`
	Sign             string   `xml:"sign"`
	Sign_type        string   `xml:"sign_type,omitempty"`
	Body             string   `xml:"body"`
	Detail           CDATA    `xml:"detail,omitempty"`
	Attach           string   `xml:"attach,omitempty"`
	Out_trade_no     string   `xml:"out_trade_no"`
	Fee_type         string   `xml:"fee_type,omitempty"`
	Total_fee        string   `xml:"total_fee"`
	Spbill_create_ip string   `xml:"spbill_create_ip"`
	Time_start       string   `xml:"time_start,omitempty"`
	Time_expire      string   `xml:"time_expire,omitempty"`
	Goods_tag        string   `xml:"goods_tag,omitempty"`
	Notify_url       string   `xml:"notify_url"`
	Trade_type       string   `xml:"trade_type"`
	Product_id       string   `xml:"product_id,omitempty"`
	Limit_pay        string   `xml:"limit_pay,omitempty"`
	Openid           string   `xml:"openid,omitempty"`
	Scene_info       CDATA    `xml:"scene_info,omitempty"`
}

type wxPayResult struct {
	Return_code  string `xml:"return_code" json:"return_code"`
	Return_msg   string `xml:"return_msg" json:"return_msg"`
	Appid        string `xml:"appid" json:"appid"`
	Mch_id       string `xml:"mch_id" json:"mch_id"`
	Device_info  string `xml:"device_info" json:"device_info"`
	Nonce_str    string `xml:"nonce_str" json:"nonce_str"`
	Sign         string `xml:"sign" json:"sign"`
	Result_code  string `xml:"result_code" json:"result_code"`
	Err_code     string `xml:"err_code" json:"err_code"`
	Err_code_des string `xml:"err_code_des" json:"err_code_des"`
	Trade_type   string `xml:"trade_type" json:"trade_type"`
	Prepay_id    string `xml:"prepay_id" json:"prepay_id"`
	Code_url     string `xml:"code_url" json:"code_url"`
}

type WechatPayServer struct {
	wechatClient
}

func NewPayServer() *WechatPayServer {
	srv := new(WechatPayServer)
	return srv
}

func (srv *WechatPayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	param, err := srv.parseParam(r)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}
	srv.signParam(param)

	result, err := srv.sendParam(param)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}

	js, err := json.Marshal(result)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}
	w.Write(js)
}

func (srv *WechatPayServer) parseParam(r *http.Request) (p *wxPayParam, err error) {
	f := r.Form
	p = new(wxPayParam)

	p.AppId = f.Get("appid")
	p.Mch_id = f.Get("mch_id")
	p.Mch_key = f.Get("mch_key")
	p.Notify_url = srv.normalizeUrl(r, f.Get("notify_url"), "")
	p.Body = f.Get("body")
	p.Out_trade_no = f.Get("out_trade_no")
	p.Total_fee = f.Get("total_fee")
	p.Spbill_create_ip = f.Get("spbill_create_ip")

	// check necessary parameters
	if p.AppId == "" || p.Mch_id == "" || p.Mch_key == "" || p.Notify_url == "" || p.Body == "" ||
		p.Out_trade_no == "" || p.Total_fee == "" || p.Spbill_create_ip == "" {
		err = errors.New("necessary parameters: appid, mch_id, mch_key, notify_url, body, out_trade_no, total_fee, spbill_create_ip")
		return
	}
	fee, err := strconv.Atoi(p.Total_fee)
	if err != nil {
		return
	}
	if fee == 0 {
		err = errors.New("total_fee should be greater than 0")
		return
	}

	// get other necessary parameters
	p.Nonce_str = randomString(32)
	p.Trade_type = "NATIVE"
	if f.Get("trade_type") != "" {
		p.Trade_type = f.Get("trade_type")
	}

	// optional parameters
	p.Device_info = f.Get("device_info")
	p.Sign_type = f.Get("sign_type")
	p.Detail = CDATA(f.Get("detail"))
	p.Attach = f.Get("attach")
	p.Fee_type = f.Get("fee_type")
	p.Time_start = f.Get("time_start")
	p.Time_expire = f.Get("time_expire")
	p.Goods_tag = f.Get("goods_tag")
	p.Product_id = f.Get("product_id")
	p.Limit_pay = f.Get("limit_pay")
	p.Openid = f.Get("openid")
	p.Scene_info = CDATA(f.Get("scene_info"))

	return
}

func (srv *WechatPayServer) signParam(p *wxPayParam) {
	var arr []string

	key := p.Mch_key
	p.Mch_key = ""
	p.Sign = ""

	t := reflect.TypeOf(*p)
	v := reflect.ValueOf(*p)
	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).Kind() != reflect.String {
			continue
		}
		name := strings.ToLower(t.Field(i).Name)
		value := v.Field(i).String()
		if value == "" {
			continue
		}
		str := fmt.Sprintf("%s=%s", name, value)
		arr = append(arr, str)
	}

	sort.Strings(arr)
	sign_str := strings.Join(arr, "&") + "&key=" + key
	hash_bytes := md5.Sum([]byte(sign_str))

	p.Sign = fmt.Sprintf("%X", hash_bytes[:])
}

func (srv *WechatPayServer) sendParam(p *wxPayParam) (r *wxPayResult, err error) {
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

	r = new(wxPayResult)
	err = xml.Unmarshal(resp_bytes, r)
	return
}
