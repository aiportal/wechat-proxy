package wxproxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	"encoding/xml"
	"encoding/json"
	"reflect"
)

const messageRequestTimeout = 5 * time.Second

// doc: https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421135319
type WechatMessageServer struct {
	wechatClient
}

func NewMessageServer() *WechatMessageServer {
	srv := new(WechatMessageServer)
	return srv
}

func (srv *WechatMessageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)
	r.ParseForm()

	if r.Method == http.MethodGet {
		echostr := r.Form.Get("echostr")
		w.Write([]byte(echostr))
		return
	}

	// parse parameters
	f := r.Form
	_, timestamp, nonce := f.Get("signature"), f.Get("timestamp"), f.Get("nonce")
	encrypt_type, msg_signature := f.Get("encrypt_type"), f.Get("msg_signature")
	token, aes_key := f.Get("token"), f.Get("aes")

	call_urls := srv.getCalls(r)

	// read body
	defer r.Body.Close()
	req_body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err.Error())
		return
	}
	log.Println(string(req_body))

	// dispatch raw message
	if token == "" || aes_key == "" || encrypt_type == "" {
		if strings.HasSuffix(r.URL.Path, "/json") && encrypt_type == "" {
			resp_body, err := srv.translateMsg(req_body, call_urls)
			if err != nil {
				log.Println(err.Error())
				return
			}
			w.Write(resp_body)
			return
		}
		resp_body := srv.dispatchMsg(req_body, call_urls)
		w.Write(resp_body)
		return
	}

	// decrypt and dispatch message
	c, err := NewCrypter(token, aes_key)
	if err != nil {
		log.Println(err.Error())
		return
	}

	msg, appid, err := c.DecryptPkg(bytes.NewReader(req_body), timestamp, nonce, msg_signature)
	if err != nil {
		log.Println(err.Error())
		return
	}
	var reply []byte
	if strings.HasSuffix(r.URL.Path, "/json") {
		reply, err = srv.translateMsg(msg, call_urls)
		if err != nil {
			log.Println(err.Error())
			return
		}
	} else {
		reply = srv.dispatchMsg(msg, call_urls)
	}

	resp_body, err := c.EncryptPkg(reply, appid)
	if err != nil {
		log.Println(err.Error())
		return
	}
	w.Write(resp_body)
}

// dispatch json message
func (srv *WechatMessageServer) translateMsg(msg []byte, urls []string) (reply []byte, err error) {
	var m wxMessage
	err = xml.Unmarshal(msg, &m)
	if err != nil {
		return
	}
	msg_js, err := json.Marshal(m)
	if err != nil {
		return
	}
	if m.MsgType == "event" {
		t := wxEventsMap[m.Event]
		if t != nil {
			n := reflect.New(wxEventsMap[m.Event])
			err = xml.Unmarshal(msg, &n)
			if err != nil {
				return
			}
			msg_js, err = json.Marshal(n)
			if err != nil {
				return
			}
		}
	}

	reply_js := srv.dispatchMsg(msg_js, urls)

	var r WxReply
	err = json.Unmarshal(reply_js, &r)
	if err != nil {
		return
	}
	reply, err = xml.Marshal(r)
	return
}

// dispatch message body to calls url
func (srv *WechatMessageServer) dispatchMsg(body []byte, urls []string) (result []byte) {

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

func (srv *WechatMessageServer) getCalls(r *http.Request) []string {
	// prepare callback urls
	calls := r.Form["call"]
	if len(calls) < 1 {
		return calls
	}
	query := srv.msgQuery(r)
	for i, v := range calls {
		calls[i] = srv.normalizeUrl(r, v, query)
	}
	return calls
}

// Get wechat message query parameters
func (srv *WechatMessageServer) msgQuery(r *http.Request) string {
	f := r.Form
	signature, timestamp, nonce := f.Get("signature"), f.Get("timestamp"), f.Get("nonce")
	query := fmt.Sprintf("signature=%s&timestamp=%s&nonce=%s", signature, timestamp, nonce)

	if r.Method == http.MethodGet {
		echostr := f.Get("echostr")
		query += fmt.Sprintf("&echostr=%s", echostr)
	} else {
		encrypt_type, msg_signature := f.Get("encrypt_type"), f.Get("msg_signature")
		query += fmt.Sprintf("&encrypt_type=%s&msg_signature=%s", encrypt_type, msg_signature)
	}
	return query
}

// Get absolute url contain http:// or https://
func (srv *WechatMessageServer) normalizeUrl(r *http.Request, url string, query string) string {
	if strings.HasPrefix(url, "/") {
		url = srv.hostUrl(r) + url
	} else if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}
	if !strings.Contains(url, "?") {
		url += "?"
	} else {
		url += "&"
	}
	return url + query
}
