package wxproxy

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115

type wxJsConfig struct {
	Appid     string   `json:"appId"`
	Secret    string   `json:"-"`
	Debug     bool     `json:"debug"`
	Timestamp uint64   `json:"timestamp"`
	NonceStr  string   `json:"nonceStr"`
	Signature string   `json:"signature"`
	JsApiList []string `json:"jsApiList"`
}

type WechatJsConfigServer struct {
	wechatClient
	configMap *cacheMap
}

func NewJsConfigServer() *WechatJsConfigServer {
	srv := new(WechatJsConfigServer)
	return srv
}

func (srv *WechatJsConfigServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	// parse
	cfg, wxErr := srv.parseParam(r)
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}

	// signature
	wxErr = srv.jsSignature(cfg, srv.hostUrl(r), r.Referer())
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}

	// return config script
	bs, err := json.Marshal(cfg)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}
	resp_body := fmt.Sprintf(`wx.config(%s)`, string(bs))
	w.Write([]byte(resp_body))
}

func (srv *WechatJsConfigServer) parseParam(r *http.Request) (cfg *wxJsConfig, wxErr *wxError) {
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")
	debug, api_list := r.Form.Get("debug"), r.Form.Get("apilist")

	// check request parameters
	if appid == "" || secret == "" {
		wxErr = wxErrorStr("parameters: appid, secret, debug(optinal), apilist(optinal)")
		return
	}

	// store config info
	cfg = new(wxJsConfig)
	cfg.Appid = appid
	cfg.Secret = secret
	if strings.EqualFold(debug, "true") {
		cfg.Debug = true
	}
	cfg.JsApiList = strings.Split(api_list, ",")

	return
}

func (srv *WechatJsConfigServer) jsSignature(cfg *wxJsConfig, hostUrl, url string) (wxErr *wxError) {
	// get jsapi_ticket
	ticket, wxErr := srv.getJsTicket(hostUrl, cfg.Appid, cfg.Secret)
	if wxErr != nil {
		return
	}

	// signature
	nonceStr := randomString(16)
	timestamp := time.Now().Unix()
	s := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticket, nonceStr, timestamp, url)
	hash := sha1.Sum([]byte(s))
	signature := fmt.Sprintf("%x", hash[:])

	cfg.NonceStr = nonceStr
	cfg.Timestamp = uint64(timestamp)
	cfg.Signature = signature

	return
}
