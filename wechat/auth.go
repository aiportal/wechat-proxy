package wechat

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"
)

const (
	authRequestDuration = 5 * time.Minute
	authRequestLimit    = 1000
)

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140842
type WechatAuthServer struct {
	WechatClient
	requestMap *CacheMap
}

func NewAuthServer() *WechatAuthServer {
	srv := new(WechatAuthServer)
	srv.requestMap = NewCacheMap(authRequestDuration, authRequestLimit)
	return srv
}

// /auth?appid=...&secret=...&call=...&lang=
// /auth/info?appid=...&secret=...&call=...&lang=
func (srv *WechatAuthServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)

	r.ParseForm()
	f := r.Form
	code, key := f.Get("code"), f.Get("key")

	// redirect to auth url
	if code == "" {
		auth_url := srv.authUrl(r)
		log.Println(auth_url)

		script := fmt.Sprintf(`<script>window.location.href="%s";</script>`, auth_url)
		w.Write([]byte(script))
		return
	}
	log.Printf("auth code: %s\n", code)

	//-------------------------------------------
	// wechat client second visit here
	//-------------------------------------------

	// load auth param
	param, ok := srv.requestMap.Get(key)
	if !ok {
		w.Write([]byte("auth timeout"))
		return
	}
	p := param.(authParam)

	info := &wxUserInfo{}
	defer srv.postInfo(w, &p, info)

	// request auth token
	t, wxErr := srv.authToken(r, &p, code)
	if wxErr != nil {
		info.WxError = *wxErr
		return
	}
	info.OpenId = t.OpenId

	// request user info
	if t.Scope == "snsapi_userinfo" {
		info_url := "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=%s"
		_url := fmt.Sprintf(info_url, t.AuthToken, t.OpenId, p.Lang)
		_, err := HttpGetJson(_url, &info)
		if err != nil {
			info.WxError = *NewError(err)
			return
		}
	} else {
		// get unionid
		access_token, wxErr := srv.GetAccessToken(srv.HostUrl(r), p.AppId, p.Secret)
		if wxErr != nil {
			info.WxError = *wxErr
			return
		}

		info_url := "https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=%s"
		_url := fmt.Sprintf(info_url, access_token, t.OpenId, p.Lang)
		_, err := HttpGetJson(_url, &info)
		if err != nil {
			info.WxError = *NewError(err)
			return
		}
	}
}

func (srv *WechatAuthServer) authUrl(r *http.Request) string {
	f := r.Form

	p := &authParam{
		AppId:  f.Get("appid"),
		Secret: f.Get("secret"),
		Call:   srv.NormalizeUrl(r, f.Get("call"), ""),
		State:  f.Get("state"),
		Lang:   f.Get("lang"),
	}

	scope := "snsapi_base"
	if strings.HasSuffix(r.URL.Path, "/info") {
		scope = "snsapi_userinfo"
	}

	path_hash := md5.Sum([]byte(r.RequestURI))
	key := fmt.Sprintf("%x", path_hash[:])

	srv.requestMap.Set(key, *p)
	defer srv.requestMap.Shrink()

	// generate auth url
	// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
	redirect_uri := fmt.Sprintf("%s%s?key=%s", srv.HostUrl(r), r.URL.Path, key)
	base_url := "https://open.weixin.qq.com/connect/oauth2/authorize"
	auth_url := fmt.Sprintf("%s?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect",
		base_url, p.AppId, url.QueryEscape(redirect_uri), scope, p.State)

	return auth_url
}

func (srv *WechatAuthServer) authToken(r *http.Request, p *authParam, code string) (t *wxAuthToken, wxErr *WxError) {

	base_url := "https://api.weixin.qq.com/sns/oauth2/access_token"
	token_url := fmt.Sprintf("%s?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		base_url, p.AppId, p.Secret, code)

	t = &wxAuthToken{}
	_, err := HttpGetJson(token_url, &t)
	if err != nil {
		wxErr = NewError(err)
		return
	}
	if !t.Success() {
		wxErr = &t.WxError
		return
	}
	return
}

func (srv *WechatAuthServer) postInfo(w http.ResponseWriter, p *authParam, info *wxUserInfo) {

	f := &authForm{
		wxUserInfo: *info,
		AppId:      p.AppId,
		State:      p.State,
		Sign:       "",
	}
	f.Sign = srv.signForm(f, p.Secret)

	bs, err := json.Marshal(f)
	if err != nil {
		return
	}
	js := string(bs)

	htmlTemplate := `
	<body onload='document.forms[0].submit()'>
	  <form method='POST' enctype='text/plain' action='%s' style='display:none;'>
		<input name='%s' value='%s'>
	  </form>
	</body>`
	pos := len(js) - 2
	html := fmt.Sprintf(htmlTemplate, p.Call, js[:pos], js[pos:])
	w.Write([]byte(html))
}

func (*WechatAuthServer) signForm(f *authForm, key string) string {
	var arr []string

	t := reflect.TypeOf(*f)
	v := reflect.ValueOf(*f)
	for i := 0; i < t.NumField(); i++ {
		if v.Field(i).Kind() != reflect.String {
			continue
		}
		name := strings.ToLower(t.Field(i).Name)
		value := v.Field(i).String()
		if value == "" {
			continue
		}
		str := fmt.Sprintf("%s=%s", name, value)
		arr = append(arr, str)
	}

	sort.Strings(arr)
	sign_str := strings.Join(arr, "&") + "&key=" + key
	hash_bytes := md5.Sum([]byte(sign_str))

	return fmt.Sprintf("%X", hash_bytes[:])
}

type authParam struct {
	AppId  string
	Secret string
	Call   string
	State  string
	Lang   string
}

type authForm struct {
	wxUserInfo
	AppId string `json:"appid"`
	State string `json:"state"`
	Sign  string `json:"sign"`
}

type wxAuthToken struct {
	WxError
	AuthToken    string `json:"access_token"`
	Expires      int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
	Scope        string `json:"scope"`
}

type wxUserInfo struct {
	WxError
	OpenId     string   `json:"openid"`
	UnionId    string   `json:"unionid"`
	NickName   string   `json:"nickname,omitempty"`
	Sex        int      `json:"sex,omitempty"`
	City       string   `json:"city,omitempty"`
	Province   string   `json:"province,omitempty"`
	Country    string   `json:"country,omitempty"`
	HeadImgUrl string   `json:"headimgurl,omitempty"`
	Privilege  []string `json:"privilege,omitempty"`
}
