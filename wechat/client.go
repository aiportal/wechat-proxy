package wechat

import (
	"fmt"
	"net/http"
	"strings"
)

type WechatClient struct {
}

func (c *WechatClient) HostUrl(r *http.Request) string {
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	return fmt.Sprintf("%s%s", scheme, r.Host)
}

func (srv *WechatClient) NormalizeUrl(r *http.Request, url string, query string) string {

	if strings.HasPrefix(url, "/") {
		url = srv.HostUrl(r) + url
	}
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	if query != "" {
		if !strings.Contains(url, "?") {
			url = url + "?" + query
		} else {
			url = url + "&" + query
		}
	}
	return url
}

func (c *WechatClient) GetAccessToken(hostUrl, appid, secret string) (accessToken string, err *WxError) {
	token_url := fmt.Sprintf("%s/api?appid=%s&secret=%s", hostUrl, appid, secret)

	var t WxAccessToken
	_, e := HttpGetJson(token_url, &t)
	if e != nil {
		err = NewError(e)
		return
	}
	if !t.Success() {
		err = &t.WxError
		return
	}
	accessToken = t.AccessToken
	return
}

func (c *WechatClient) GetJsTicket(hostUrl, appid, secret string) (jsTicket string, err *WxError) {
	ticket_url := fmt.Sprintf("%s/jsapi?appid=%s&secret=%s", hostUrl, appid, secret)

	var t wxJsTicket
	_, e := HttpGetJson(ticket_url, &t)
	if e != nil {
		err = NewError(e)
		return
	}
	if !t.Success() {
		err = &t.WxError
		return
	}
	jsTicket = t.Ticket
	return
}
