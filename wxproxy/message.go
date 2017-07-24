package wxproxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const messageRequestTimeout = 5 * time.Second

const messageContentType = "application/x-www-form-urlencoded"

var emptyStringBytes = []byte("")

type MessageServer struct {
}

func NewMessageServer() *MessageServer {
	srv := new(MessageServer)
	return srv
}

func (srv *MessageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.RequestURI)
	r.ParseForm()

	// prepare callback urls
	call_urls := r.Form["call"]
	if len(call_urls) < 1 {
		w.Write(emptyStringBytes)
		return
	}
	query := srv.messageQuery(&r.Form)
	for i, v := range call_urls {
		call_urls[i] = srv.normalizeUrl(v, query)
		fmt.Printf("call: %s\n", call_urls[i])
	}

	// verify callback urls
	if r.Method == http.MethodGet {
		echostr := r.Form.Get("echostr")
		verify := srv.verifyCallback(call_urls, echostr)
		if verify {
			w.Write([]byte(echostr))
		} else {
			w.Write(emptyStringBytes)
		}
		return
	}

	// dispatch callback message
	defer r.Body.Close()
	data := srv.dispatchMessage(call_urls, r.Body)
	if len(data) > 0 {
		w.Write(data)
	} else {
		w.Write(emptyStringBytes)
	}
}

// Check all callback url.
func (srv *MessageServer) verifyCallback(urls []string, echostr string) (success bool) {

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

func (srv *MessageServer) dispatchMessage(urls []string, body io.Reader) (result []byte) {

	chs := make([]chan []byte, len(urls))
	for i, _url := range urls {
		chs[i] = make(chan []byte)

		go func(url string, data io.Reader, ch chan []byte) {
			defer close(ch)

			client := &http.Client{
				Timeout: messageRequestTimeout,
			}
			resp, err := client.Post(url, messageContentType, body)
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
func (srv *MessageServer) messageQuery(form *url.Values) string {
	signature, timestamp, nonce := form.Get("signature"), form.Get("timestamp"), form.Get("nonce")
	echostr := form.Get("echostr")
	if signature == "" {
		signature = form.Get("msg_signature")
	}
	query := fmt.Sprintf("signature=%s&timestamp=%s&nonce=%s", signature, timestamp, nonce)
	if echostr != "" {
		query += fmt.Sprintf("&echostr=%s", echostr)
	}
	return query
}

// Get absolute url contain http:// or https://
func (srv *MessageServer) normalizeUrl(url string, query string) string {
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
