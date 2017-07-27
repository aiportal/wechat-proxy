package wxproxy

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// doc: https://work.weixin.qq.com/api/doc#10013
type WechatQYApiServer struct {
	tokenMap *cacheMap
}

func NewQYApiServer() *WechatQYApiServer {
	srv := new(WechatQYApiServer)
	srv.tokenMap = NewCacheMap(TokenCacheDuration, TokenCacheLimit)
	return srv
}

func (srv *WechatQYApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("corpid"), r.Form.Get("corpsecret")
	hashBytes := md5.Sum([]byte(appid + ":" + secret))
	hashKey := string(hashBytes[:])

	if value, ok := srv.tokenMap.Get(hashKey); ok {
		w.Write([]byte(value.(string)))
		return
	}

	_url := srv.accessTokenUrl(appid, secret)
	body, err := srv.httpGet(_url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(body) < 10 {
		w.Write([]byte(`{"errcode":40001,"errmsg":"invalid credential"}`))
		return
	}
	err = srv.parseError(body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(body)
	srv.tokenMap.Set(hashKey, string(body))
	srv.tokenMap.Shrink()
	return
}

// https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET
func (srv *WechatQYApiServer) accessTokenUrl(appid, secret string) string {
	baseUrl := "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	_url := fmt.Sprintf("%s?corpid=%s&corpsecret=%s", baseUrl, appid, secret)
	return _url
}

func (srv *WechatQYApiServer) httpGet(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}

func (srv *WechatQYApiServer) parseError(data []byte) (err error) {
	var e wxError
	err = json.Unmarshal(data, &e)
	if err != nil {
		return
	}
	if e.ErrCode != 0 {
		err = errors.New(string(data))
	}
	return
}
