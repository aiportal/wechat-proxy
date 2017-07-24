package wxproxy

import (
	"net/http"
)

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140183
// https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET
const apiBaseUrl = "https://api.weixin.qq.com/cgi-bin/token"

type ApiServer struct {
	tokenMap *cacheMap
}

func NewApiServer() *ApiServer {
	srv := new(ApiServer)
	srv.tokenMap = NewCacheMap(TokenCacheDuration, TokenCacheLimit)
	return srv
}

func (srv *ApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid := r.Form.Get("appid")

	if value, ok := srv.tokenMap.Get(appid); ok {
		w.Write([]byte(value))
		return
	}

	url := apiBaseUrl + "?" + r.URL.RawQuery
	body, err := httpGet(url)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	err = parseError(body)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(body)
	srv.tokenMap.Set(appid, string(body))
	srv.tokenMap.Shrink()
	return
}
