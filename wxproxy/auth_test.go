package wxproxy

import (
	"testing"
	"net/http"
	"io/ioutil"
	"fmt"
	"log"
	"strings"
	"os"
)

func TestAuthServer(t *testing.T) {
	if !argsContainsAuth() {
		return
	}

	log.SetOutput(ioutil.Discard)

	hostUrl := "http://debug.ultragis.com"
	appid := "wx06766a90ab72960e"
	secret := "05bd8b6064a9941b72ee44d5b3bfdb6a"
	openid := "o3u5it9y28OQoXLXh-s-EPhY3xu8"

	type AuthInfo struct {
		AuthUrl		string	`json:"auth_uri"`
		Expires		uint32	`json:"expires_in"`
	}

	// get auth proxy url
	_url := hostUrl + fmt.Sprintf("/auth?appid=%s&secret=%s&redirect_uri=%s", appid, secret, hostUrl + "/echo")
	ts_urls := []string{
		_url,
		_url + "&scope=snsapi_userinfo",
	}

	auth_urls := []string{"", ""}
	for i, _url := range ts_urls {
		var info = &AuthInfo{}
		body, err := httpGetJson(_url, info)
		if err != nil {
			fmt.Println("body: " + string(body))
			t.Fatal(err)
		}
		auth_urls[i] = info.AuthUrl
	}

	// send test message
	wxClient := &wechatClient{}
	access_token, err := wxClient.getAccessToken(hostUrl, appid, secret)
	if err != nil {
		fmt.Println(err.String())
		t.Fatal(err)
	}

	err1 := sendTestAuthLink(access_token, openid, auth_urls[0], auth_urls[1])
	if err1 != nil {
		t.Fatal(err1)
	}
}

func argsContainsAuth() bool {
	for _, a := range os.Args {
		if a == "auth" {
			return true
		}
	}
	return false
}

func sendTestAuthLink(accessToken, openid, link_base, link_info string) (err error) {
	sendUrl := "https://api.weixin.qq.com/cgi-bin/message/custom/send"
	send_url := fmt.Sprintf("%s?access_token=%s", sendUrl, accessToken)

	data := fmt.Sprintf(`{
	"touser":"%s",
	"msgtype":"news",
	"news":{
        "articles": [
         {
             "title":"Wechat-Proxy Project",
             "url":"https://github.com/aiportal/wechat-proxy/blob/master/README.md",
			 "picurl":"https://raw.githubusercontent.com/aiportal/wechat-proxy/master/_doc/auth_test_icon.png"
         },
         {
             "title":"snsapi_base auth test",
             "url":"%s",
  			 "picurl":"https://assets-cdn.github.com/images/modules/logos_page/Octocat.png"
         },
         {
             "title":"snsapi_userinfo auth test",
             "url":"%s",
             "picurl":"https://assets-cdn.github.com/images/modules/logos_page/Octocat.png"
         },
         ]
	}}`, openid, link_base, link_info)

	resp, err := http.Post(send_url, "", strings.NewReader(data))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	fmt.Println(string(body))

	return
}
