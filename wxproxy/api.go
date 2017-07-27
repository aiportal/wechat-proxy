package wxproxy

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140183
type WechatApiServer struct {
	tokenMap *cacheMap
}

func NewApiServer() *WechatApiServer {
	srv := new(WechatApiServer)
	srv.tokenMap = NewCacheMap(TokenCacheDuration, TokenCacheLimit)
	return srv
}

func (srv *WechatApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")
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

// url: https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET
func (srv *WechatApiServer) accessTokenUrl(appid, secret string) string {
	baseUrl := "https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential"
	_url := fmt.Sprintf("%s&appid=%s&secret=%s", baseUrl, appid, secret)
	return _url
}

func (srv *WechatApiServer) httpGet(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}

func (srv *WechatApiServer) parseError(data []byte) (err error) {
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
