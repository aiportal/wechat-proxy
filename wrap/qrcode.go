package wrap

import (
	qrcode "github.com/skip2/go-qrcode"
	"log"
	"net/http"
	"strconv"
	"wechat-proxy/wechat"
)

type QrCodeServer struct {
}

func NewQrCodeServer() *QrCodeServer {
	return &QrCodeServer{}
}

func (srv *QrCodeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	f := r.Form

	path := f.Get("path")
	log.Printf("qrcode: %s\n", path)

	size, _ := strconv.Atoi(f.Get("size"))
	if size == 0 {
		size = 256
	}

	bs, err := qrcode.Encode(path, qrcode.Medium, size)
	if err != nil {
		w.Write(wechat.JsonResponse(err))
		return
	}
	w.Write(bs)
	w.Header().Set("Content-Type", "image/png")
}
