package wrap

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	wx "wechat-proxy/wechat"
)

type WrapAppServer struct {
	wx.WechatClient
}

func NewWrapAppServer() *WrapAppServer {
	srv := new(WrapAppServer)
	return srv
}

func (srv *WrapAppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)

	// parse path
	req_path := strings.TrimLeft(r.URL.Path, "/")
	parts := strings.Split(req_path, "/")
	if len(parts) < 3 {
		http.NotFound(w, r)
		return
	}
	key := parts[1]
	path := "/" + strings.Join(parts[2:], "/")

	// load app
	app, err := NewStorage().LoadApp(key)
	if err != nil {
		log.Println(err.Error())
		http.NotFound(w, r)
		return
	}
	if app == nil {
		http.NotFound(w, r)
		return
	}

	// check path in calls
	if !app.inCalls(path) {
		log.Printf("not in calls: %s\n", path)
		http.NotFound(w, r)
		return
	}

	// generate api url
	url := srv.realUrl(r, path, app)
	log.Println(url)

	// call api
	err = srv.httpProxy(w, r, url)
	if err != nil {
		w.Write(wx.JsonResponse(err))
		return
	}
}

func (srv *WrapAppServer) realUrl(r *http.Request, path string, app *WxApp) string {

	// generate api url
	query := r.URL.RawQuery
	if strings.HasPrefix(path, "/msg") {
		query += fmt.Sprintf("&appid=%s&secret=%s&token=%s&aes=%s", app.AppId, app.Secret, app.Token, app.AesKey)
	} else if strings.HasPrefix(path, "/pay") {
		query += fmt.Sprintf("&appid=%s&mch_id=%s&mch_key=%s&server_ip=%s",
			app.AppId, app.MchId, app.MchKey, app.IpAddress)
	} else {
		query += fmt.Sprintf("&appid=%s&secret=%s", app.AppId, app.Secret)
	}

	_url := fmt.Sprintf("%s%s?%s", srv.HostUrl(r), path, query)
	return _url
}

func (srv *WrapAppServer) httpProxy(w http.ResponseWriter, r *http.Request, url string) (err error) {
	log.Println(url)

	defer r.Body.Close()
	req, err := http.NewRequest(r.Method, url, r.Body)
	if err != nil {
		return
	}
	for k, v := range r.Header {
		req.Header[k] = v
	}
	for _, c := range r.Cookies() {
		req.AddCookie(c)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	for k := range resp.Header {
		v := resp.Header.Get(k)
		w.Header().Set(k, v)
	}
	w.Write(body)
	return
}
