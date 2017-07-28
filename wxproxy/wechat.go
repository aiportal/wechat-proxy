package wxproxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type WechatUserInfo struct {
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

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140839
type WechatClient struct {
	hostUrl string
	appid   string
	secret  string
}

func NewWechatClient(hostUrl, appid, secret string) *WechatClient {
	var wx = new(WechatClient)
	wx.hostUrl = hostUrl
	wx.appid = appid
	wx.secret = secret
	return wx
}

func (c *WechatClient) getAccessToken() (token string, err error) {
	token_url := fmt.Sprintf("%s/api?appid=%s&secret=%s", c.hostUrl, c.appid, c.secret)
	resp, err := http.Get(token_url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var tokenResult struct {
		AccessToken string `json:"access_token"`
		Expires     uint64 `json:"expires_in"`
	}
	err = json.Unmarshal(body, &tokenResult)
	if err != nil {
		return
	}
	token = tokenResult.AccessToken
	return
}

func (c *WechatClient) getUserInfo(openid, lang string) (info *WechatUserInfo, err error) {
	access_token, err := c.getAccessToken()
	if err != nil {
		return
	}
	userinfo_url := "https://api.weixin.qq.com/cgi-bin/user/info?access_token=%s&openid=%s&lang=%s"
	_url := fmt.Sprintf(userinfo_url, access_token, openid, lang)

	info = new(WechatUserInfo)
	err = c.getJsonObject(_url, &info)
	if err != nil {

		return
	}
	return
}

func (c *WechatClient) getJsonObject(url string, obj interface{}) (err error) {
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
	if err != nil {
		fmt.Println(url)
		fmt.Println(string(body))
		return
	}
	return
}
