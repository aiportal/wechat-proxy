package wrap

import (
	"log"
	"net/http"
	wx "wechat-proxy/wechat"
)

type RegisterServer struct {
	wx.WechatClient
}

func NewRegisterServer() *RegisterServer {
	return &RegisterServer{}
}

func (srv *RegisterServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println(r.RequestURI)
	r.ParseForm()
	f := r.Form

	// get parameters
	app := &WxApp{
		Key:       f.Get("key"),
		AppId:     f.Get("appid"),
		Secret:    f.Get("secret"),
		Token:     f.Get("token"),
		MchId:     f.Get("mch_id"),
		MchKey:    f.Get("mch_key"),
		IpAddress: f.Get("server_ip"),
		AesKey:    f.Get("aes"),
	}
	app.setCalls(f["call"])

	// set expires time
	err := app.setExpires(f.Get("expires"))
	if err != nil {
		w.Write(wx.JsonResponse(err))
		return
	}

	// check appid and secret
	_, wxErr := srv.GetAccessToken(srv.HostUrl(r), app.AppId, app.Secret)
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}

	// store app info
	err = NewStorage().SaveApp(app)
	//err = srv.storePermanent(app)
	if err != nil {
		w.Write(wx.JsonResponse(err))
		return
	}
	w.Write(wx.JsonResponse(nil))
	return
}
