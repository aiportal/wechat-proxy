package wxproxy

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	authRequestDuration = 5 * time.Minute
	authRequestLimit    = 1000
)

var ErrAuthTimeout = errors.New("auth timeout")

type authRequestParam struct {
	HostUrl  string
	AppId    string
	Secret   string
	Redirect string
	Scope    string
	State    string
	Lang     string
}

type wxAuthToken struct {
	wxError
	AuthToken    string `json:"access_token"`
	Expires      int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
}

type wxUserInfo struct {
	wxError
	OpenId  string `json:"openid"`
	UnionId string `json:"unionid"`

	NickName   string   `json:"nickname,omitempty"`
	Sex        int      `json:"sex,omitempty"`
	Province   string   `json:"province,omitempty"`
	City       string   `json:"city,omitempty"`
	Country    string   `json:"country,omitempty"`
	HeadImgUrl string   `json:"headimgurl,omitempty"`
	Privilege  []string `json:"privilege,omitempty"`
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
type WechatAuthServer struct {
	wechatClient
	requestMap *cacheMap
}

func NewAuthServer() *WechatAuthServer {
	srv := new(WechatAuthServer)
	srv.requestMap = NewCacheMap(authRequestDuration, authRequestLimit)
	return srv
}

// Wechat auth process:
// Client call url: https://wx.ultragis.com/auth?appid=APPID&secret=SECRET
// &redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
// At response, client get an short url to show.
// If user visit the short url by wechat, wechat will redirect to redirect_uri by post data.
// The post data is encrypted by secret and contains openid, unionid and access_token, refresh_token
// Then, client can get userinfo by access_token and refresh_token(depend on scope).
func (srv *WechatAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	//-------------------------------------------
	// init auth proxy
	//-------------------------------------------

	authid := r.Form.Get("authid")
	if authid == "" {
		// check and parse parameters
		param, err := srv.parseParameters(r)
		if err != nil {
			log.Println(err.Error())
			return
		}
		// return proxy url
		proxy_url, duration := srv.generateProxyUrl(r, param)
		proxy_json := fmt.Sprintf(`{"auth_uri":"%s", "expires_in":%d}`, proxy_url, duration)
		w.Write([]byte(proxy_json))
		return
	}

	//-------------------------------------------
	// wechat client first visit here
	//-------------------------------------------

	// load request parameters from cache map
	param, ok := srv.requestMap.Get(authid)
	if !ok {
		log.Println(ErrAuthTimeout)
		return
	}
	p := param.(authRequestParam)

	code := r.Form.Get("code")
	if code == "" {
		// redirect wechat oauth2 url
		auth_url := srv.wechatAuthUrl(&p, authid)
		http.Redirect(w, r, auth_url, http.StatusTemporaryRedirect)
		log.Printf("auth_url: %s\n", auth_url)
		return
	}

	//-------------------------------------------
	// wechat client second visit here
	//-------------------------------------------
	var info *wxUserInfo
	var err error
	if p.Scope == "snsapi_base" {
		info, err = srv.getBaseInfo(&p, code)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
	if p.Scope == "snsapi_userinfo" {
		info, err = srv.getUserInfo(&p, code)
		if err != nil {
			log.Println(err.Error())
			return
		}
	}
	html, err := srv.postForm(&p, info)
	if err != nil {
		log.Println(err.Error())
		return
	}
	w.Write([]byte(html))
	return
}

// parse and check request parameters
func (srv *WechatAuthServer) parseParameters(r *http.Request) (p *authRequestParam, err error) {
	f := r.Form

	if f.Get("appid") == "" || f.Get("secret") == "" || f.Get("redirect_uri") == "" {
		tip := "need parameters: appid, secret, redirect_uri, scope(optional), state(optional), lang(optional)"
		err = errors.New(tip)
		return
	}

	redirect_uri := f.Get("redirect_uri")
	redirect_uri = srv.normalizeUrl(redirect_uri, "")
	_, err = url.Parse(redirect_uri)
	if err != nil {
		return
	}

	scope := f.Get("scope")
	if scope != "snsapi_base" && scope != "snsapi_userinfo" {
		scope = "snsapi_base"
	}

	lang := f.Get("lang")
	if lang == "" {
		lang = "zh_CN"
	}

	p = new(authRequestParam)
	p.HostUrl = srv.hostUrl(r)
	p.AppId = f.Get("appid")
	p.Secret = f.Get("secret")
	p.Redirect = redirect_uri
	p.Scope = scope
	p.State = f.Get("state")
	p.Lang = lang

	return
}

// generate proxy url by authid
func (srv *WechatAuthServer) generateProxyUrl(r *http.Request, p *authRequestParam) (proxyUrl string, duration uint64) {

	// Store request parameters in cache
	query_rand := fmt.Sprintf("%s&_=%d", r.URL.RawQuery, time.Now().Unix())
	query_hash := md5.Sum([]byte(query_rand))
	authid := fmt.Sprintf("%X", query_hash[:])
	srv.requestMap.Set(authid, *p)
	defer srv.requestMap.Shrink()

	// generate proxy url
	proxyUrl = fmt.Sprintf("%s/auth?authid=%s", srv.hostUrl(r), authid)
	duration = uint64(authRequestDuration / time.Second)
	return
}

// generate wechat oauth2 url.
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://open.weixin.qq.com/connect/oauth2/authorize?
// 		appid=APPID&redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
func (srv *WechatAuthServer) wechatAuthUrl(p *authRequestParam, authid string) string {

	redirect_uri := fmt.Sprintf("%s/auth?authid=%s", p.HostUrl, authid)

	baseUrl := "https://open.weixin.qq.com/connect/oauth2/authorize"
	_url := fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		baseUrl, p.AppId, url.QueryEscape(redirect_uri), p.Scope, p.State)
	return _url
}

func (srv *WechatAuthServer) wechatTokenUrl(p *authRequestParam, code string) string {
	baseUrl := "https://api.weixin.qq.com/sns/oauth2/access_token"
	_url := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		baseUrl, p.AppId, p.Secret, code)
	return _url
}

func (srv *WechatAuthServer) getBaseInfo(p *authRequestParam, code string) (info *wxUserInfo, err error) {
	var t wxAuthToken
	token_url := srv.wechatTokenUrl(p, code)
	_, err = httpGetJson(token_url, &t)
	if err != nil {
		return
	}
	if !t.Success() {
		err = errors.New(t.wxError.String())
		return
	}
	log.Printf("auth openid: %s\n", t.OpenId)

	access_token, e := srv.getAccessToken(p.HostUrl, p.AppId, p.Secret)
	if e != nil {
		err = errors.New(e.String())
		return
	}
	info_url := "https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=%s"
	_url := fmt.Sprintf(info_url , access_token, t.OpenId, p.Lang)

	info = new(wxUserInfo)
	_, err = httpGetJson(_url, &info)
	return
}

func (srv *WechatAuthServer) getUserInfo(p *authRequestParam, code string) (info *wxUserInfo, err error) {
	var t wxAuthToken
	token_url := srv.wechatTokenUrl(p, code)
	_, err = httpGetJson(token_url, &t)
	if err != nil {
		return
	}
	if !t.Success() {
		err = errors.New(t.wxError.String())
		return
	}
	log.Printf("auth openid: %s\n", t.OpenId)

	//return srv.requestUserInfo(t.AuthToken, t.OpenId, p.Lang)
	detail_url := "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=%s"
	_url := fmt.Sprintf(detail_url, t.AuthToken, t.OpenId, p.Lang)

	info = new(wxUserInfo)
	_, err = httpGetJson(_url, &info)
	return
}

// client open a html page and post data to server automatic
// ref: https://systemoverlord.com/2016/08/24/posting-json-with-an-html-form.html
func (srv *WechatAuthServer) postForm(p *authRequestParam, info *wxUserInfo) (html string, err error) {

	type FormInfo struct {
		wxUserInfo
		AppId string `json:"appid"`
		State string `json:"state"`
		Trash string `json:"trash"`
	}
	var f = FormInfo{
		wxUserInfo: *info,
		AppId:          p.AppId,
		State:          p.State,
	}

	jsonData, err := json.Marshal(f)
	if err != nil {
		return
	}
	jsonStr := string(jsonData)
	// fmt.Printf("post json: %s\n", jsonStr)

	htmlTemplate := `
	<body onload='document.forms[0].submit()'>
	  <form method='POST' enctype='text/plain' action='%s' style='display:none;'>
		<input name='%s' value='%s'>
	  </form>
	</body>`
	pos := len(jsonStr) - 2
	html = fmt.Sprintf(htmlTemplate, p.Redirect, jsonStr[:pos], jsonStr[pos:])
	return
}

// Get absolute url contain http:// or https://
func (srv *WechatAuthServer) normalizeUrl(url string, query string) string {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	if query == "" {
		return url
	}
	if !strings.Contains(url, "?") {
		url += "?"
	} else {
		url += "&"
	}
	return url + query
}
