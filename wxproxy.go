package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"wechat-proxy/enterprise"
	"wechat-proxy/wechat"
	"wechat-proxy/wrap"
)

func main() {
	wrapHandlers()
	wechatHandlers()
	enterpriseHandlers()

	http.Handle("/example/", http.StripPrefix("/example/", http.FileServer(http.Dir("./example"))))
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(body)
		log.Println(string(body))
	})

	host, port, tls := parseArgs()
	address := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("wechat proxy starting at %q ...\n", address)

	if tls {
		log.Fatal(http.ListenAndServeTLS(address, "wxproxy.crt", "wxproxy.key", nil))
	} else {
		log.Fatal(http.ListenAndServe(address, nil))
	}
}

func parseArgs() (host string, port uint, tls bool) {

	flag.StringVar(&host, "host", "", "Listening hostname.")
	flag.UintVar(&port, "port", 8080, "Listening port.")
	flag.UintVar(&port, "p", 8080, "Listening port.")
	flag.BoolVar(&tls, "tls", false, "Https scheme.")

	flag.Parse()
	return
}

func wrapHandlers() {

	// /register?key=...&appid=...&secret=...
	// &token=&aes=
	// &mch_id=&mch_key=&server_ip=
	// &expires=&call=/msg&call=/api&call=
	http.Handle("/register", wrap.NewRegisterServer())

	// /app/<key>/api
	// /app/<key>/msg?signature=...
	// ...
	http.Handle("/app/", wrap.NewWrapAppServer())

	// /qrcode?path=...&size=
	http.Handle("/qrcode", wrap.NewQrCodeServer())

	// /short?path=...&expires=
	// http.Handle("/short/", wrap.NewShortServer())

	// /user
	// userServer := wrap.NewUserServer()
	// http.Handle("/user", userServer)
	// http.Handle("/user/", userServer)
}

func wechatHandlers() {

	// /api?appid=...&secret=...
	// /api/new?appid=...&secret=...
	apiServer := wechat.NewApiServer()
	http.Handle("/api", apiServer)
	http.Handle("/api/new", apiServer)

	// /msg?token=...&aes=...&call=...&call=...&signature=...&...
	// /msg/json?token=...&aes=...&call=...&call=...&signature=...&...
	msgServer := wechat.NewMessageServer()
	http.Handle("/msg", msgServer)
	http.Handle("/msg/json", msgServer)

	// /auth??appid=...&secret=...&call=...&state=&lang=
	// /auth/info?appid=...&secret=...&call=...&state=&lang=
	authServer := wechat.NewAuthServer()
	http.Handle("/auth", authServer)      // get openid & unionid
	http.Handle("/auth/info", authServer) // get user info

	payServer := wechat.NewPayServer()
	http.Handle("/pay", payServer)
	// /pay/qrcode?
	// &appid=...&mch_id=...&mch_key=...&server_ip=...
	// &fee=...&name=&call=&...
	http.Handle("/pay/qrcode", payServer)
	// /pay/js?
	// &appid=...&mch_id=...&mch_key=...&server_ip=...
	// &openid=...&fee=...&name=&call=&...
	http.Handle("/pay/js", payServer)

	// /jsapi?appid=...&secret=...
	// /jsapi?access_token=...
	http.Handle("/jsapi", wechat.NewJsTicketServer())

	// /js/config?appid=...&secret=...&debug=&apilist=
	http.Handle("/js/config", wechat.NewJsConfigServer())

	http.Handle("/js/card", wechat.NewCardServer())
}

func enterpriseHandlers() {

	// /qyapi?corpid=...&corpsecret=...
	// /qyapi/new?corpid=...&corpsecret=...
	qyServer := enterprise.NewQyServer()
	http.Handle("/qyapi", qyServer)
	http.Handle("/qyapi/new", qyServer)
}
