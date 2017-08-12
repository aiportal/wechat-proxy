package wxproxy

import (
	"fmt"
	"net/http"
)

type wxJsTicket struct {
	wxError
	Ticket  string `json:"ticket"`
	Expires uint32 `json:"expires_in"`
}

type WechatJsServer struct {
	wechatClient
	ticketMap *cacheMap
}

func NewJsServer() *WechatJsServer {
	srv := new(WechatJsServer)
	srv.ticketMap = NewCacheMap(tokenCacheDuration, tokenCacheLimit)
	return srv
}

func (srv *WechatJsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")
	access_token := r.Form.Get("access_token")

	if access_token == "" {
		var err *wxError
		access_token, err = srv.getAccessToken(srv.hostUrl(r), appid, secret)
		if err != nil {
			w.Write(err.Serialize())
			return
		}
	}

	// try get ticket
	if value, ok := srv.ticketMap.Get(access_token); ok {
		w.Write(value.([]byte))
		return
	}

	jsapi_base_url := "https://api.weixin.qq.com/cgi-bin/ticket/getticket"
	_url := fmt.Sprintf("%s?access_token=%s&type=jsapi", jsapi_base_url, access_token)
	var t wxJsTicket
	body, err := httpGetJson(_url, &t)
	if err != nil {
		w.Write(newError(err).Serialize())
		return
	}
	if !t.Success() {
		w.Write(t.Serialize())
		return
	}

	w.Write(body)
	srv.ticketMap.Set(access_token, body)
	go srv.ticketMap.Shrink()
	return
}
