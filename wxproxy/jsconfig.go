package wxproxy

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115

const (
	jsConfigDuration = 48 * time.Hour
	jsConfigLimit    = 500
)

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
	srv.configMap = NewCacheMap(jsConfigDuration, jsConfigLimit)
	return srv
}

// server get config_uri,
// then client request config_uri
func (srv *WechatJsConfigServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	config_key := r.Form.Get("key")

	//-------------------------------------------------------------------------
	// Server request config_uri by appid and secret
	//-------------------------------------------------------------------------
	if config_key == "" {
		key, err := srv.cacheParameters(r)
		if err != nil {
			w.Write(err.Serialize())
			return
		}
		config_url := fmt.Sprintf("%s%s?key=%s", srv.hostUrl(r), r.URL.Path, key)
		duration := jsConfigDuration / time.Second
		js := fmt.Sprintf(`{"config_uri":"%s", "expires_in":%d}`, config_url, duration)
		w.Write([]byte(js))
		return
	}

	//-------------------------------------------------------------------------
	// Client request js by config_uri
	//-------------------------------------------------------------------------
	value, ok := srv.configMap.Get(config_key)
	if !ok {
		log.Println("jsconfig timeout")
		return
	}

	// signature
	cfg := value.(*wxJsConfig)
	wxErr := srv.jsSignature(cfg, srv.hostUrl(r), r.Referer())
	if wxErr != nil {
		log.Println(wxErr.String())
		return
	}

	// return config script
	bs, err := json.Marshal(cfg)
	if err != nil {
		log.Println(err.Error())
		return
	}
	resp_body := fmt.Sprintf(`wx.config(%s)`, string(bs))
	w.Write([]byte(resp_body))
}

func (srv *WechatJsConfigServer) cacheParameters(r *http.Request) (key string, wxErr *wxError) {
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")
	debug, api_list := r.Form.Get("debug"), r.Form.Get("apilist")

	// check request parameters
	if appid == "" || secret == "" {
		wxErr = wxErrorStr("parameters: appid, secret, debug(optinal), apilist(optinal)")
		return
	}
	_, wxErr = srv.getJsTicket(srv.hostUrl(r), appid, secret)
	if wxErr != nil {
		return
	}

	// store config info
	cfg := new(wxJsConfig)
	cfg.Appid = appid
	cfg.Secret = secret
	if strings.EqualFold(debug, "true") {
		cfg.Debug = true
	}
	cfg.JsApiList = strings.Split(api_list, ",")

	key = srv.hashKey(r.URL.RawQuery)
	srv.configMap.Set(key, cfg)
	go srv.configMap.Shrink()

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

func (srv *WechatJsConfigServer) hashKey(query string) string {
	query_hash := md5.Sum([]byte(query))
	return fmt.Sprintf("%x", query_hash[:])
}
