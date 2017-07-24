package wxproxy

import (
	"net/http"
)

// https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET
const qyapiBaseUrl = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"

type QYApiServer struct {
	tokenMap *cacheMap
}

func NewQYApiServer() *QYApiServer {
	srv := new(QYApiServer)
	srv.tokenMap = NewCacheMap(TokenCacheDuration, TokenCacheLimit)
	return srv
}

func (srv *QYApiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	corpid := r.Form.Get("corpid")

	if value, ok := srv.tokenMap.Get(corpid); ok {
		w.Write([]byte(value))
		return
	}

	url := qyapiBaseUrl + "?" + r.URL.RawQuery
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
	srv.tokenMap.Set(corpid, string(body))
	srv.tokenMap.Shrink()
	return
}
