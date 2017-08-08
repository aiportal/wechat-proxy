package wxproxy

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	// access_token stay time in memory
	TokenCacheDuration = 3600 * time.Second

	// access_token max count in memory
	TokenCacheLimit = 100

	// secret stay time in memory
	SecretCacheDuration = 48 * 3600 * time.Second

	// secret max count in memory
	SecretCacheLimit = 100
)

type wxAccessToken struct {
	wxError
	AccessToken string `json:"access_token"`
	ExpiresIn   uint64 `json:"expires_in"`
}

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140183
type WechatApiServer struct {
	tokenMap *cacheMap
	secretMap *cacheMap
}

func NewApiServer() *WechatApiServer {
	srv := new(WechatApiServer)
	srv.tokenMap = NewCacheMap(TokenCacheDuration, TokenCacheLimit)
	srv.secretMap = NewCacheMap(SecretCacheDuration, SecretCacheLimit)
	return srv
}

func (srv *WechatApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")

	// find token
	hashBytes := md5.Sum([]byte(appid + ":" + secret))
	hashKey := string(hashBytes[:])
	if value, ok := srv.tokenMap.Get(hashKey); ok {
		w.Write(value.([]byte))
		return
	}

	token := new(wxAccessToken)
	_url := srv.accessTokenUrl(appid, secret)
	body, err := srv.httpGetJson(_url, token)
	if err != nil {
		w.Write([]byte(newError(err).String()))
		return
	}
	if !token.Success() {
		w.Write([]byte(token.wxError.String()))
		return
	}

	w.Write(body)
	srv.tokenMap.Set(hashKey, body)
	srv.tokenMap.Shrink()
	srv.secretMap.Set(appid, secret)
	srv.secretMap.Shrink()
	return
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

func (srv *WechatApiServer) getSecret(appid string) (secret string, ok bool) {
	s, ok := srv.secretMap.Get(appid)
	if !ok {
		return
	}
	secret, ok = s.(string)
	return
}

func (srv *WechatApiServer) getAccessToken(appid string) (accessToken string, ok bool) {
	s, ok := srv.secretMap.Get(appid)
	if !ok {
		return
	}
	secret, ok := s.(string)
	if !ok {
		return
	}
	hashBytes := md5.Sum([]byte(appid + ":" + secret))
	hashKey := string(hashBytes[:])
	t, ok := srv.tokenMap.Get(hashKey)
	if !ok {
		return
	}

	token := new(wxAccessToken)
	err := json.Unmarshal(t.([]byte), token)
	if err != nil {
		ok = false
		return
	}
	accessToken = token.AccessToken
	return
}
