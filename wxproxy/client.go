package wxproxy

import (
	"net/http"
	"fmt"
)

type wechatClient struct {
}

func (c *wechatClient) hostUrl(r *http.Request) string {
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return fmt.Sprintf("%s%s", scheme, r.Host)
}

func (c *wechatClient) getAccessToken(hostUrl, appid, secret string) (accessToken string, err *wxError) {
	token_url := fmt.Sprintf("%s/api?appid=%s&secret=%s", hostUrl, appid, secret)

	var t wxAccessToken
	_, e := httpGetJson(token_url, &t)
	if e != nil {
		err = newError(e)
		return
	}
	if !t.Success() {
		err = &t.wxError
		return
	}
	accessToken = t.AccessToken
	return
}

func (c *wechatClient) getJsTicket(hostUrl, appid, secret string) (jsTicket string, err *wxError) {
	ticket_url := fmt.Sprintf("%s/jsapi?appid=%s&secret=%s", hostUrl, appid, secret)

	var t wxJsTicket
	_, e := httpGetJson(ticket_url, &t)
	if e != nil {
		err = newError(e)
		return
	}
	if !t.Success() {
		err = &t.wxError
		return
	}
	jsTicket = t.Ticket
	return
}
