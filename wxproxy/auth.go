package wxproxy

import (
	"crypto/md5"
	"encoding/json"
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
	Appid    string
	Secret   string
	Redirect string
	Scope    string
	State    string
}

type AuthTokenInfo struct {
	wxError
	AccessToken  string `json:"access_token"`
	Expires      int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
}

type AuthResultUserInfo struct {
	wxError
	OpenId  string `json:"openid"`
	UnionId string `json:"unionid"`

	NickName   string   `json:"nickname"`
	Sex        string   `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgUrl string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
}

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
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
	fmt.Println(r.RequestURI)
	r.ParseForm()

	//-------------------------------------------
	// init auth proxy
	//-------------------------------------------

	authid := r.Form.Get("authid")
	if authid == "" {
		// return proxy url, description by json
		proxy_url, duration := srv.proxyUrl(srv.rawUrl(r))
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
		w.WriteHeader(http.StatusNotFound)
		return
	}
	p := param.(AuthRequestParam)

	code := r.Form.Get("code")
	if code == "" {
		// redirect wechat oauth2 url
		auth_url := srv.wechatAuthUrl(&p)
		http.Redirect(w, r, auth_url, http.StatusTemporaryRedirect)
		return
	}

	//-------------------------------------------
	// wechat client second visit here
	//-------------------------------------------

	// Get openid from wechat server
	token := srv.requestTokenInfo(&p, code)
	if !token.Success() {
		info := AuthResultUserInfo{wxError: token.wxError}
		srv.clientPost(&p, &info)
		return
	}

	// get user info from wechat server
	if p.Scope == "snsapi_base" {
		// request unionid
		info := srv.requestBaseInfo(&p, token.OpenId)
		srv.clientPost(&p, info)
		return
	}

	if p.Scope == "snsapi_userinfo" {
		// request user info
		info := srv.requestUserInfo(&p, token.OpenId, token.AccessToken)
		srv.clientPost(&p, info)
		return
	}
}

// get raw url
func (srv *WechatAuthServer) rawUrl(r *http.Request) *url.URL {
	scheme := "http://"
	if r.TLS != nil {
		scheme = "https://"
	}
	raw_url := strings.Join([]string{scheme, r.Host, r.RequestURI}, "")
	_url, _ := url.Parse(raw_url)
	return _url
}

// generate a json package contains proxy url.
func (srv *WechatAuthServer) proxyUrl(_url *url.URL) (proxyUrl string, duration uint64) {

	// parse auth request parameters
	form := _url.Query()
	param := AuthRequestParam{
		Appid:    form.Get("appid"),
		Secret:   form.Get("secret"),
		Redirect: form.Get("redirect_uri"),
		Scope:    form.Get("scope"),
		State:    form.Get("state"),
	}
	param.Redirect = srv.normalizeUrl(param.Redirect, "")

	// Store request parameters in cache
	auth_query := fmt.Sprintf("%s&t=%d", _url.RawQuery, time.Now().Unix())
	auth_hash := md5.Sum([]byte(auth_query))
	authid := fmt.Sprintf("%X", auth_hash[:])
	srv.requestMap.Set(authid, param)
	defer srv.requestMap.Shrink()

	// generate proxy url
	proxyUrl = fmt.Sprintf("%s://%s%s?authid=%s", _url.Scheme, _url.Host, _url.Path, authid)
	duration = uint64(AuthRequestDuration / time.Second)
	return
}

// generate wechat oauth2 url.
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://open.weixin.qq.com/connect/oauth2/authorize?
// 		appid=APPID&redirect_uri=REDIRECT_URI&response_type=code&scope=SCOPE&state=STATE#wechat_redirect
func (srv *WechatAuthServer) wechatAuthUrl(p *AuthRequestParam) string {

	baseUrl := "https://open.weixin.qq.com/connect/oauth2/authorize"
	_url := fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		baseUrl, p.Appid, p.Redirect, p.Scope, p.State)
	return _url
}

// generate wechat oauth2 url for: use code in exchange for access_token.
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://api.weixin.qq.com/sns/oauth2/access_token?
// 		appid=APPID&secret=SECRET&code=CODE&grant_type=authorization_code
func (srv *WechatAuthServer) requestTokenInfo(p *AuthRequestParam, code string) (token *AuthTokenInfo) {
	baseUrl := "https://api.weixin.qq.com/sns/oauth2/access_token"
	_url := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		baseUrl, p.Appid, p.Secret, code)

	token = new(AuthTokenInfo)
	err := srv.getJsonObject(_url, &token)
	if err != nil {
		token.wxError = wxError{ ErrCode:-10001, ErrMsg:err.Error() }
		return token
	}
	return token
}

// request unionid by service app access_token and openid
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://api.weixin.qq.com/cgi-bin/user/info?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
func (srv *WechatAuthServer) requestBaseInfo(p *AuthRequestParam, openid string) *AuthResultUserInfo {
	// TODO: ...

	info := new(AuthResultUserInfo)
	return info
}

// request user info by oauth2 access_token and openid
// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
// url: https://api.weixin.qq.com/sns/userinfo?access_token=ACCESS_TOKEN&openid=OPENID&lang=zh_CN
func (srv *WechatAuthServer) requestUserInfo(p *AuthRequestParam, openid, token string) *AuthResultUserInfo {
	// TODO: ...

	info := new(AuthResultUserInfo)
	return info
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

// Post data to redirectUrl
// ref: https://systemoverlord.com/2016/08/24/posting-json-with-an-html-form.html
func (srv *WechatAuthServer) redirectForm(redirectUrl string, postBody string) string {
	htmlTemplate := `
	<body onload='document.forms[0].submit()'>
	  <form method='POST' enctype='text/plain' action='%s' style='display:none;'>
		<input name='%s' value='%s'>
	  </form>
	</body>`
	html := fmt.Sprintf(htmlTemplate, redirectUrl, postBody, "")
	return html
}

// client open a html page and post data to server automatic
func (srv *WechatAuthServer) clientPost(p *AuthRequestParam, info *AuthResultUserInfo) {
	secret := []byte(p.Secret)
	if len(secret) != 32 {
		emptyBytes := [32]byte{}
		secret = append(secret, emptyBytes[:]...)
		secret = secret[:32]
	}
	//e, _ := aes.NewCipher([]byte(p.Secret))
	//for k, v := range m {
	//	dst := new([]byte, len(v))
	//	e.Encrypt(dst, []byte(v))
	//	m[k] = fmt.Sprintf("%x", dst)
	//}
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
