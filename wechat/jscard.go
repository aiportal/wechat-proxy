package wechat

import (
	"fmt"
	"net/http"
)

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421141115
type wxCardTicket struct {
	WxError
	Ticket  string `json:"ticket"`
	Expires uint32 `json:"expires_in"`
}

type WechatCardServer struct {
	WechatClient
	ticketMap *CacheMap
}

func NewCardServer() *WechatCardServer {
	srv := new(WechatCardServer)
	srv.ticketMap = NewCacheMap(tokenCacheDuration, tokenCacheLimit)
	return srv
}

func (srv *WechatCardServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	appid, secret := r.Form.Get("appid"), r.Form.Get("secret")
	access_token := r.Form.Get("access_token")

	if access_token == "" {
		var err *WxError
		access_token, err = srv.GetAccessToken(srv.HostUrl(r), appid, secret)
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

	card_base_url := "https://api.weixin.qq.com/cgi-bin/ticket/getticket"
	_url := fmt.Sprintf("%s?access_token=%s&type=wx_card", card_base_url, access_token)
	var t wxJsTicket
	body, err := HttpGetJson(_url, &t)
	if err != nil {
		w.Write(NewError(err).Serialize())
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
