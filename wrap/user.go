package wrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	wx "wechat-proxy/wechat"
)

type WechatUserServer struct {
	wx.WechatClient
}

func NewUserServer() *WechatUserServer {
	srv := &WechatUserServer{}
	return srv
}

func (srv *WechatUserServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)

	if r.Method == http.MethodGet {

		// query user info by appid and other conditions

		return
	}

	msg, err := srv.getMessage(r, "event", "subscribe", "unsubscribe", "LOCATION")
	if err != nil {
		log.Println(err.Error())
		return
	}
	if msg == nil {
		return
	}

	r.ParseForm()

	if msg.Event == "subscribe" {
		go srv.subscribe(r, msg)
	}
	if msg.Event == "unsubscribe" {
		go srv.unsubscribe(r, msg)
	}
	if msg.Event == "LOCATION" {
		go srv.location(r, msg)
	}
}

func (*WechatUserServer) getMessage(r *http.Request, msgType string, events ...string) (msg *wx.WxMessage, err error) {

	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	m := &wx.WxMessage{}
	err = json.Unmarshal(body, m)
	if err != nil {
		return
	}

	if m.MsgType != msgType {
		return
	}

	for _, v := range events {
		if strings.ToLower(m.Event) == strings.ToLower(v) {
			msg = m
			return
		}
	}
	return
}

func (srv *WechatUserServer) subscribe(r *http.Request, m *wx.WxMessage) {
	log.Printf("subscribe: %#v\n", m)

	f := r.Form
	appid := f.Get("appid")
	secret := f.Get("secret")
	openid := m.FromUserName

	info, err := srv.getUserInfo(r, appid, secret, openid)
	if err != nil {
		log.Println(err.Error())
		return
	}

	key := strings.TrimPrefix(m.EventKey, "qrscene_")
	u := &WxUser{
		AppId:           appid,
		OpenId:          openid,
		UnionId:         info.Unionid,
		Subscribe:       true,
		SubscribeTime:   m.CreateTime,
		UnSubscribeTime: 0,
		Referral:        key,

		Nickname:   info.Nickname,
		Sex:        info.Sex,
		City:       info.City,
		Country:    info.Country,
		Province:   info.Province,
		Language:   info.Language,
		HeadImgUrl: info.HeadImgUrl,
		Remark:     info.Remark,
	}

	err = NewStorage().SaveUser(u)
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func (*WechatUserServer) unsubscribe(r *http.Request, m *wx.WxMessage) {
	log.Printf("unsubscribe: %#v\n", m)

	f := r.Form
	appid := f.Get("appid")

	openid := m.FromUserName
	u, err := NewStorage().LoadUser(appid, openid)
	if err != nil {
		log.Println(err.Error())
		return
	}

	u.Subscribe = false
	u.UnSubscribeTime = m.CreateTime

	err = NewStorage().SaveUser(u)
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func (*WechatUserServer) location(r *http.Request, m *wx.WxMessage) {
	log.Printf("location: %#v\n", m)

	f := r.Form
	appid := f.Get("appid")

	openid := m.FromUserName
	u, err := NewStorage().LoadUser(appid, openid)
	if err != nil {
		log.Println(err.Error())
		return
	}

	u.Latitude = m.Latitude
	u.Longitude = m.Longitude
	u.Precision = m.Precision
	u.LocationTime = m.CreateTime
}

func (srv *WechatUserServer) getUserInfo(r *http.Request, appid, secret, openid string) (u *wxUserInfo, err error) {
	access_token, wxErr := srv.GetAccessToken(srv.HostUrl(r), appid, secret)
	if wxErr != nil {
		err = errors.New(wxErr.ErrMsg)
		return
	}

	url_base := "https://api.weixin.qq.com/cgi-bin/user/info"
	_url := fmt.Sprintf("%s?access_token=%s&openid=%s&lang=zh_CN", url_base, access_token, openid)
	u = &wxUserInfo{}
	_, err = wx.HttpGetJson(_url, u)
	if !u.Success() {
		log.Printf("user info: %s\n", _url)
		return
	}
	return
}

type wxUserInfo struct {
	wx.WxError
	Openid        string `json:"openid"`         // 用户的标识，对当前公众号唯一
	Unionid       string `json:"unionid"`        // 只有在用户将公众号绑定到微信开放平台帐号后，才会出现该字段。
	Subscribe     int    `json:"subscribe"`      // 用户是否订阅该公众号标识，值为0时，代表此用户没有关注该公众号，拉取不到其余信息。
	SubscribeTime uint64 `json:"subscribe_time"` // 用户关注时间，为时间戳。如果用户曾多次关注，则取最后关注时间
	Nickname      string `json:"nickname"`       // 用户的昵称
	Sex           int `json:"sex"`            // 用户的性别，值为1时是男性，值为2时是女性，值为0时是未知
	City          string `json:"city"`           // 用户所在城市
	Country       string `json:"country"`        // 用户所在国家
	Province      string `json:"province"`       // 用户所在省份
	Language      string `json:"language"`       // 用户的语言，简体中文为zh_CN
	HeadImgUrl    string `json:"headimgurl"`     // 用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），用户没有头像时该项为空。若用户更换头像，原有头像URL将失效。
	Remark        string `json:"remark"`         // 公众号运营者对粉丝的备注，公众号运营者可在微信公众平台用户管理界面对粉丝添加备注
	GroupId       int    `json:"groupid"`        // 用户所在的分组（兼容旧的用户分组接口）
	TagIdList     []int  `json:"tagid_list"`     // 用户被打上的标签
}
