package wxproxy

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var AuthRequestDuration = 300 * time.Second
var AuthRequestLimit = 1000

type AuthRequestParam struct {
	HostUrl  string
	AppId    string
	Secret   string
	Redirect string
	Scope    string
	State    string
	Lang     string
}

type AuthTokenInfo struct {
	wxError
	AccessToken  string `json:"access_token"`
	Expires      int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
type WechatAuthServer struct {
	requestMap *cacheMap
}

func NewAuthServer() *WechatAuthServer {
	srv := new(WechatAuthServer)
	srv.requestMap = NewCacheMap(AuthRequestDuration, AuthRequestLimit)
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
	//fmt.Println(r.RequestURI)
	r.ParseForm()

	//-------------------------------------------
	// init auth proxy
	//-------------------------------------------

	authid := r.Form.Get("authid")
	if authid == "" {
		// check and parse parameters
		param, err := srv.parseParameters(r)
		if err != nil {
			e := wxError{ErrCode: -10001, ErrMsg: err.Error()}
			w.Write([]byte(e.Error()))
			return
		}
		// return proxy url
		proxy_url, duration := srv.proxyUrl(r, param)
		proxy_json := fmt.Sprintf(`{"auth_uri":"%s", "expires_in":%d}`, proxy_url, duration)
		w.Write([]byte(proxy_json))

		if r.Form.Get("test") != "" {
			srv.sendTestLink(r, param, proxy_url)
		}
		return
	}

	//-------------------------------------------
	// wechat client first visit here
	//-------------------------------------------

	// load request parameters from cache map
	param, ok := srv.requestMap.Get(authid)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	p := param.(AuthRequestParam)

	code := r.Form.Get("code")
	if code == "" {
		// redirect wechat oauth2 url
		auth_url := srv.wechatAuthUrl(&p, authid)
		http.Redirect(w, r, auth_url, http.StatusTemporaryRedirect)
		fmt.Printf("auth_url: %s\n", auth_url)
		return
	}

	//-------------------------------------------
	// wechat client second visit here
	//-------------------------------------------
	fmt.Printf("code: %s\n", code)

	// Get openid from wechat server
	token := srv.requestTokenInfo(&p, code)
	if !token.Success() {
		//info := WechatUserInfo{wxError: token.wxError}
		//srv.clientPost(&p, &info)
		w.Write([]byte(token.wxError.Error()))
		return
	}
	fmt.Printf("openid: %s\n", token.OpenId)

	// get user info from wechat server
	if p.Scope == "snsapi_base" {
		fmt.Printf("scope: %s\n", p.Scope)
		// request unionid
		c := NewWechatClient(p.HostUrl, p.AppId, p.Secret)
		info, err := c.getUserInfo(token.OpenId, p.Lang)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		html, err := srv.postForm(&p, info)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(html))
		return
	}

	if p.Scope == "snsapi_userinfo" {
		fmt.Printf("scope: %s\n", p.Scope)
		// request user info
		info, err := srv.requestUserInfo(token.AccessToken, token.OpenId, p.Lang)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}

		html, err := srv.postForm(&p, info)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write([]byte(html))
		return
	}
}

// parse and check request parameters
func (srv *WechatAuthServer) parseParameters(r *http.Request) (p *AuthRequestParam, err error) {
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

	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}

	p = new(AuthRequestParam)
	p.HostUrl = fmt.Sprintf("%s%s", scheme, r.Host)
	p.AppId = f.Get("appid")
	p.Secret = f.Get("secret")
	p.Redirect = redirect_uri
	p.Scope = scope
	p.State = f.Get("state")
	p.Lang = lang

	return
}

// generate proxy url by authid
func (srv *WechatAuthServer) proxyUrl(r *http.Request, p *AuthRequestParam) (proxyUrl string, duration uint64) {

	// Store request parameters in cache
	query_rand := fmt.Sprintf("%s&_=%d", r.URL.RawQuery, time.Now().Unix())
	query_hash := md5.Sum([]byte(query_rand))
	authid := fmt.Sprintf("%X", query_hash[:])
	srv.requestMap.Set(authid, *p)
	defer srv.requestMap.Shrink()

	// generate proxy url
	proxyUrl = fmt.Sprintf("%s/auth?authid=%s", p.HostUrl, authid)
	duration = uint64(AuthRequestDuration / time.Second)
	return
}

// generate wechat oauth2 url.
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://open.weixin.qq.com/connect/oauth2/authorize?
// 		appid=APPID&redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
func (srv *WechatAuthServer) wechatAuthUrl(p *AuthRequestParam, authid string) string {

	redirect_uri := fmt.Sprintf("%s/auth?authid=%s", p.HostUrl, authid)

	baseUrl := "https://open.weixin.qq.com/connect/oauth2/authorize"
	_url := fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		baseUrl, p.AppId, url.QueryEscape(redirect_uri), p.Scope, p.State)
	return _url
}

// generate wechat oauth2 url for: use code in exchange for access_token.
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://api.weixin.qq.com/sns/oauth2/access_token?
// 		appid=APPID&secret=SECRET&code=CODE&grant_type=authorization_code
func (srv *WechatAuthServer) requestTokenInfo(p *AuthRequestParam, code string) (token *AuthTokenInfo) {
	baseUrl := "https://api.weixin.qq.com/sns/oauth2/access_token"
	_url := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		baseUrl, p.AppId, p.Secret, code)

	token = new(AuthTokenInfo)
	err := srv.getJsonObject(_url, &token)
	if err != nil {
		token.wxError = wxError{ErrCode: -10002, ErrMsg: err.Error()}
		return token
	}
	return token
}

// request user info by oauth2 access_token and openid
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://api.weixin.qq.com/sns/userinfo?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
func (srv *WechatAuthServer) requestUserInfo(token, openid, lang string) (info *WechatUserInfo, err error) {

	fmt_url := "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=%s"
	_url := fmt.Sprintf(fmt_url, token, openid, lang)

	info = new(WechatUserInfo)
	err = srv.getJsonObject(_url, info)
	if err != nil {
		return
	}
	return
}

// client open a html page and post data to server automatic
// ref: https://systemoverlord.com/2016/08/24/posting-json-with-an-html-form.html
func (srv *WechatAuthServer) postForm(p *AuthRequestParam, info *WechatUserInfo) (html string, err error) {

	type FormInfo struct {
		WechatUserInfo
		AppId string `json:"appid"`
		State string `json:"state"`
		Trash string `json:"trash"`
	}
	var f = FormInfo{
		WechatUserInfo: *info,
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

// request json object
func (srv *WechatAuthServer) getJsonObject(url string, obj interface{}) (err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, obj)
	return
}

func (srv *WechatAuthServer) sendTestLink(r *http.Request, p *AuthRequestParam, link string) {

	appid := r.Form.Get("appid")
	secret := r.Form.Get("secret")
	openid := r.Form.Get("test")

	var getWechatToken = func() (token string, err error) {
		scheme := "http://"
		if r.TLS != nil {
			scheme = "https://"
		}
		token_url := fmt.Sprintf("%s%s/api?appid=%s&secret=%s", scheme, r.Host, appid, secret)
		resp, err := http.Get(token_url)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		var tokenResult struct {
			AccessToken string `json:"access_token"`
			Expires     uint64 `json:"expires_in"`
		}
		err = json.Unmarshal(body, &tokenResult)
		if err != nil {
			fmt.Println(token_url)
			fmt.Println(string(body))
			fmt.Println(err.Error())
			return
		}
		token = tokenResult.AccessToken
		return
	}

	var sendWechatMessage = func(accessToken string) {
		sendUrl := "https://api.weixin.qq.com/cgi-bin/message/custom/send"
		send_url := fmt.Sprintf("%s?access_token=%s", sendUrl, accessToken)

		data := fmt.Sprintf(`{
	"touser":"%s",
	"msgtype":"news",
	"news":{
        "articles": [
         {
             "title":"Wechat Auth Test",
             "description":"Wechat auth test from wechat-proxy",
             "url":"%s",
             "picurl":"https://raw.githubusercontent.com/aiportal/wechat-proxy/master/WeChat-Proxy.png"
         },
         {
             "title":"Github Project",
             "description":"Github address for wechat-proxy",
             "url":"https://github.com/aiportal/wechat-proxy/blob/master/README.md",
             "picurl":"https://assets-cdn.github.com/images/modules/logos_page/Octocat.png"
         }]
	}}`, openid, link)

		resp, err := http.Post(send_url, "", strings.NewReader(data))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		fmt.Println(string(body))
	}

	access_token, err := getWechatToken()
	if err != nil {
		return
	}
	fmt.Printf("test: %s\n", link)
	sendWechatMessage(access_token)
}
