package wechat

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
	WechatClient
	configMap *CacheMap
}

func NewJsConfigServer() *WechatJsConfigServer {
	srv := new(WechatJsConfigServer)
	return srv
}

func (srv *WechatJsConfigServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	f := r.Form

	// parse
	config := &wxJsConfig{
		Appid: f.Get("appid"),
		Secret: f.Get("secret"),
		Debug: strings.EqualFold(f.Get("debug"), "true"),
		JsApiList: wxApiList,
	}
	if f.Get("apilist") != "" {
		config.JsApiList = strings.Split(f.Get("apilist"), ",")
	}

	// signature
	wxErr := srv.jsSignature(config, srv.HostUrl(r), r.Referer())
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}

	// return config script
	bs, err := json.Marshal(config)
	if err != nil {
		w.Write(NewError(err).Serialize())
		return
	}
	resp_body := fmt.Sprintf(`wx.config(%s)`, string(bs))
	w.Write([]byte(resp_body))
}

func (srv *WechatJsConfigServer) jsSignature(cfg *wxJsConfig, hostUrl, url string) (wxErr *WxError) {
	// get jsapi_ticket
	ticket, wxErr := srv.GetJsTicket(hostUrl, cfg.Appid, cfg.Secret)
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

var wxApiList = []string{
	"onMenuShareTimeline",
	"onMenuShareAppMessage",
	"onMenuShareQQ",
	"onMenuShareWeibo",
	"onMenuShareQZone",
	"startRecord",
	"stopRecord",
	"onVoiceRecordEnd",
	"playVoice",
	"pauseVoice",
	"stopVoice",
	"onVoicePlayEnd",
	"uploadVoice",
	"downloadVoice",
	"chooseImage",
	"previewImage",
	"uploadImage",
	"downloadImage",
	"translateVoice",
	"getNetworkType",
	"openLocation",
	"getLocation",
	"hideOptionMenu",
	"showOptionMenu",
	"hideMenuItems",
	"showMenuItems",
	"hideAllNonBaseMenuItem",
	"showAllNonBaseMenuItem",
	"closeWindow",
	"scanQRCode",
	"chooseWXPay",
	"openProductSpecificView",
	"addCard",
	"chooseCard",
	"openCard",
}