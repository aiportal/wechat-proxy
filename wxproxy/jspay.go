package wxproxy

import (
	"net/http"
	"fmt"
	"time"
	"strings"
	"sort"
	"crypto/md5"
	"encoding/json"
	"log"
)

type wxPayJs struct {
	Timestamp uint64 `json:"timestamp"`
	NonceStr  string `json:"nonceStr"`
	Package   string `json:"package"`
	SignType  string `json:"signType"`
	PaySign   string `json:"paySign"`
}

type WechatJsPayServer struct {
	wechatClient
}

func NewJsPayServer() *WechatJsPayServer {
	srv := new(WechatJsPayServer)
	return srv
}

func (srv *WechatJsPayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)
	r.ParseForm()

	var wxErr *wxError
	prepay_id, wxErr := srv.getPrepayId(r)
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}
	log.Printf("prepay_id: %s\n", prepay_id)

	appid := r.Form.Get("appid")
	mch_key := r.Form.Get("mch_key")
	sign := srv.paySignature(appid, prepay_id, mch_key)

	bs, err := json.Marshal(sign)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}
	w.Write(bs)
}

func (srv *WechatJsPayServer) getPrepayId(r *http.Request) (prepayId string, wxErr *wxError) {
	pay_url := fmt.Sprintf("%s/pay?%s", srv.hostUrl(r), r.URL.RawQuery)

	var pay wxPayResult
	_, err := httpGetJson(pay_url, &pay)
	if err != nil {
		wxErr = newError(err)
		return
	}
	if pay.Return_code != payResultSuccess {
		wxErr = wxErrorStr(pay.Return_msg)
		return
	}
	if pay.Result_code != payResultSuccess {
		wxErr = wxErrorStr(pay.Err_code_des)
		return
	}
	prepayId = pay.Prepay_id
	return
}

func (srv *WechatJsPayServer) paySignature(appid, prepayId, mchKey string) (sign *wxPayJs) {
	sign = new(wxPayJs)
	sign.Timestamp = uint64(time.Now().Unix())
	sign.NonceStr = randomString(32)
	sign.Package = fmt.Sprintf("prepay_id=%s", prepayId)
	sign.SignType = "MD5"

	arr := []string {
		fmt.Sprintf("appId=%s", appid),
		fmt.Sprintf("timeStamp=%d", sign.Timestamp),
		fmt.Sprintf("nonceStr=%s", sign.NonceStr),
		fmt.Sprintf("package=%s", sign.Package),
		fmt.Sprintf("signType=%s", sign.SignType),
	}
	sort.Strings(arr)
	str := strings.Join(arr, "&")
	str += fmt.Sprintf("key=%s", mchKey)

	hash := md5.Sum([]byte(str))
	sign.PaySign = fmt.Sprintf("%X", hash)
	return
}

