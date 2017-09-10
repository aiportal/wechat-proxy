package enterprise

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
	wx "wechat-proxy/wechat"
)

const (
	// access_token stay time in memory
	tokenCacheDuration = time.Hour

	// access_token max count in memory
	tokenCacheLimit = 100
)

// doc: https://work.weixin.qq.com/api/doc#10013
type WechatQyServer struct {
	tokenMap *wx.CacheMap
}

func NewQyServer() *WechatQyServer {
	srv := new(WechatQyServer)
	srv.tokenMap = wx.NewCacheMap(tokenCacheDuration, tokenCacheLimit)
	return srv
}

func (srv *WechatQyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("corpid"), r.Form.Get("corpsecret")

	key := srv.hashKey(appid, secret)
	if strings.HasSuffix(r.URL.Path, "/new") {
		srv.tokenMap.Remove(key)
	}
	if value, ok := srv.tokenMap.Get(key); ok {
		w.Write(value.([]byte))
		return
	}

	token := &wx.WxAccessToken{}
	_url := srv.accessTokenUrl(appid, secret)
	body, err := srv.httpGetJson(_url, token)
	if err != nil {
		w.Write([]byte(wx.NewError(err).String()))
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

func (srv *WechatQyServer) hashKey(appid, secret string) string {
	hashBytes := md5.Sum([]byte(appid + ":" + secret))
	return string(hashBytes[:])
}

// https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET
func (srv *WechatQyServer) accessTokenUrl(appid, secret string) string {
	baseUrl := "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	_url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", baseUrl, appid, secret)
	return _url
}

func (srv *WechatQyServer) httpGetJson(url string, obj interface{}) (body []byte, err error) {
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
