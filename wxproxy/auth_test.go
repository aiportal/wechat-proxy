package wxproxy

import (
	"testing"
	"net/http"
	"io/ioutil"
	"fmt"
	"log"
	"strings"
)

func TestAuthServer(t *testing.T) {
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
		//fmt.Println(info.AuthUrl)
		auth_urls[i] = info.AuthUrl
	}

	// send test message
	wxClient := NewWechatClient(hostUrl, appid, secret)
	access_token, err := wxClient.getAccessToken()
	if err != nil {
		fmt.Println()
		t.Fatal(err)
	}
	sendTestAuthLink(access_token, openid, auth_urls[0], auth_urls[1])
}

func sendTestAuthLink(accessToken, openid, link_base, link_info string) {
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
			 "picurl":"https://raw.githubusercontent.com/aiportal/wechat-proxy/master/WeChat-Proxy.png"
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

//
//func TestAuthInit(t *testing.T) {
//
//	ts_init := []struct{
//		Url string
//		Redirect string
//		Fields []string
//	}{
//		{
//			Url: "/auth?appid=wx000&secret=xxxxxx",
//			Redirect: "/echo",
//			Fields:[]string{"auth_uri", "expires_in"},
//		},
//		{
//			Url: "/auth?appid=wx000",
//			Redirect: "/echo",
//			Fields:[]string{"errcode", "errmsg"},
//		},
//	}
//
//	ts := httptest.NewServer(NewAuthServer())
//	defer ts.Close()
//
//	for _, v := range ts_init {
//		_url := fmt.Sprintf("%s%s&redirect_uri=%s", ts.URL, v.Url, url.PathEscape(ts.URL+v.Redirect))
//		resp, err := http.Get(_url)
//		if err != nil {
//			log.Fatal(err)
//		}
//		fmt.Println(_url)
//
//		body, err := ioutil.ReadAll(resp.Body)
//		resp.Body.Close()
//		if err != nil {
//			log.Fatal(err)
//		}
//		//fmt.Printf("body: %s\n", string(body))
//
//		var f interface{}
//		err = json.Unmarshal(body, &f)
//		if err != nil {
//			fmt.Printf("url: %s\n", _url)
//			fmt.Printf("body: %s\n", string(body))
//			log.Fatal(err)
//		}
//
//		m := f.(map[string]interface{})
//		for _, fld := range v.Fields {
//			if m[fld] == nil {
//				log.Fatal()
//			}
//		}
//	}
//}
