package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"wechat-proxy/wxproxy"
)

func main() {

	http.Handle("/api", wxproxy.NewApiServer())
	http.Handle("/qyapi", wxproxy.NewQYApiServer())
	http.Handle("/svc", wxproxy.NewMessageServer())
	http.Handle("/auth", wxproxy.NewAuthServer())

	//http.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
	//	r.ParseForm()
	//	echostr := r.Form.Get("echostr")
	//	w.Write([]byte(echostr))
	//})

	host, port := parseArgs()
	address := fmt.Sprintf("%s:%d", host, port)

	fmt.Printf("wechat proxy starting at %q ...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func parseArgs() (host string, port uint) {

	flag.StringVar(&host, "host", "", "Listening hostname.")
	flag.UintVar(&port, "port", 8080, "Listening port.")

	flag.Parse()
	return
}
