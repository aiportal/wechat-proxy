package wechat

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

const (
	// access_token stay time in memory
	tokenCacheDuration = 3600 * time.Second

	// access_token max count in memory
	tokenCacheLimit = 100
)

type WxAccessToken struct {
	WxError
	AccessToken string `json:"access_token"`
	ExpiresIn   uint64 `json:"expires_in"`
}

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140183
type WechatApiServer struct {
	tokenMap *CacheMap
}

func NewApiServer() *WechatApiServer {
	srv := new(WechatApiServer)
	srv.tokenMap = NewCacheMap(tokenCacheDuration, tokenCacheLimit)
	return srv
}

func (srv *WechatApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")

	// find token
	key := srv.hashKey(appid, secret)
	if strings.HasSuffix(r.URL.Path, "/new") {
		srv.tokenMap.Remove(key)
	}
	if value, ok := srv.tokenMap.Get(key); ok {
		w.Write(value.([]byte))
		return
	}

	token := &WxAccessToken{}
	_url := srv.accessTokenUrl(appid, secret)
	body, err := srv.httpGetJson(_url, token)
	if err != nil {
		w.Write([]byte(NewError(err).String()))
		return
	}
	if !token.Success() {
		w.Write([]byte(token.WxError.String()))
		return
	}

	w.Write(body)
	srv.tokenMap.Set(key, body)
	go srv.tokenMap.Shrink()
	return
}

func (srv *WechatApiServer) hashKey(appid, secret string) string {
	hashBytes := md5.Sum([]byte(appid + ":" + secret))
	return string(hashBytes[:])
}

// url: https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET
func (srv *WechatApiServer) accessTokenUrl(appid, secret string) string {
	baseUrl := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential"
	_url := fmt.Sprintf("%s&appid=%s&secret=%s", baseUrl, appid, secret)
	return _url
}

func (srv *WechatApiServer) httpGetJson(url string, obj interface{}) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, obj)
	return
}
