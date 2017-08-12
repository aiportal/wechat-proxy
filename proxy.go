package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"wechat-proxy/wxproxy"
)

func main() {

	http.Handle("/api", wxproxy.NewApiServer())
	http.Handle("/qyapi", wxproxy.NewQyServer())
	http.Handle("/msg", wxproxy.NewMessageServer())
	http.Handle("/auth", wxproxy.NewAuthServer())
	http.Handle("/js", wxproxy.NewJsServer())
	http.Handle("/card", wxproxy.NewCardServer())

	//http.Handle("/crypto", wxproxy.NewCryptoServer())
	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		w.Write(body)
	})

	host, port := parseArgs()

	address := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("wechat proxy starting at %q ...\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func parseArgs() (host string, port uint) {

	flag.StringVar(&host, "host", "", "Listening hostname.")
	flag.StringVar(&host, "h", "", "Listening hostname.")
	flag.UintVar(&port, "port", 8080, "Listening port.")
	flag.UintVar(&port, "p", 8080, "Listening port.")

	flag.Parse()
	return
}
