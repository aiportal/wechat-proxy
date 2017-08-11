package wxproxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const messageRequestTimeout = 5 * time.Second

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421135319
type WechatMessageServer struct {
}

func NewMessageServer() *WechatMessageServer {
	srv := new(WechatMessageServer)
	return srv
}

func (srv *WechatMessageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)
	r.ParseForm()

	// prepare callback urls
	call_urls := r.Form["call"]
	if len(call_urls) < 1 {
		return
	}
	query := srv.messageQuery(&r.Form)
	for i, v := range call_urls {
		call_urls[i] = srv.normalizeUrl(v, query)
		log.Printf("call: %s\n", call_urls[i])
	}

	// verify callback urls
	if r.Method == http.MethodGet {
		echostr := r.Form.Get("echostr")
		verify := srv.verifyCallback(call_urls, echostr)
		if verify {
			w.Write([]byte(echostr))
		}
		return
	}

	// dispatch callback message
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	data := srv.dispatchMessage(call_urls, body)
	w.Write(data)
}

// Check all callback url.
func (srv *WechatMessageServer) verifyCallback(urls []string, echostr string) (success bool) {

	chs := make([]chan string, len(urls))
	for i, _url := range urls {
		chs[i] = make(chan string)

		go func(url string, ch chan string) {
			defer close(ch)

			client := &http.Client{
				Timeout: messageRequestTimeout,
			}
			resp, err := client.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return
			}
			data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			ch <- string(data)
		}(_url, chs[i])
	}

	success = true
	for _, ch := range chs {
		result := <-ch
		if result != echostr {
			success = false
		}
	}
	return
}

// dispatch message body to calls define
func (srv *WechatMessageServer) dispatchMessage(urls []string, body []byte) (result []byte) {

	chs := make([]chan []byte, len(urls))
	for i, _url := range urls {
		chs[i] = make(chan []byte)

		go func(url string, data []byte, ch chan []byte) {
			defer close(ch)

			client := &http.Client{
				Timeout: messageRequestTimeout,
			}
			resp, err := client.Post(url, "", bytes.NewReader(data))
			if err != nil {
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return
			}
			resp_data, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			ch <- resp_data
		}(_url, body, chs[i])
	}

	for _, ch := range chs {
		result = <-ch
		if len(result) > 0 {
			break
		}
	}
	return
}

// Get wechat message query parameters
func (srv *WechatMessageServer) messageQuery(form *url.Values) string {
	signature, timestamp, nonce := form.Get("signature"), form.Get("timestamp"), form.Get("nonce")
	query := fmt.Sprintf("signature=%s&timestamp=%s&nonce=%s", signature, timestamp, nonce)

	echostr := form.Get("echostr")
	if echostr != "" {
		query += fmt.Sprintf("&echostr=%s", echostr)
	} else {
		encrypt_type, msg_signature := form.Get("encrypt_type"), form.Get("msg_signature")
		query += fmt.Sprintf("&encrypt_type=%s&msg_signature=%s", encrypt_type, msg_signature)
	}
	return query
}

// Get absolute url contain http:// or https://
func (srv *WechatMessageServer) normalizeUrl(url string, query string) string {
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	if !strings.Contains(url, "?") {
		url += "?"
	} else {
		url += "&"
	}
	return url + query
}
