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

	// get necessary parameters
	key, appid, secret := f.Get("key"), f.Get("appid"), f.Get("secret")

	// check appid and secret
	_, wxErr := srv.GetAccessToken(srv.HostUrl(r), appid, secret)
	if wxErr != nil {
		w.Write(wxErr.Serialize())
		return
	}

	// check key and appid
	app, err := NewStorage().LoadApp(key)
	if err != nil {
		app = &WxApp{
			Key: key,
			AppId: appid,
			Secret: secret,
		}
	} else {
		if app.AppId != appid {
			wxErr = wx.NewErrorStr("key exists")
			w.Write(wxErr.Serialize())
			return
		}
	}

	// merge parameters
	if f.Get("token") != "" { 
		app.Token = f.Get("token")
	}
	if f.Get("aes") != "" {
		app.AesKey = f.Get("aes")
	}
	if f.Get("mch_id") != "" {
		app.MchId = f.Get("mch_id")
	}
	if f.Get("mch_key") != "" {
		app.MchKey = f.Get("mch_key")
	}
	if f.Get("server_ip") != "" {
		app.IpAddress = f.Get("server_ip")
	}
	if f.Get("call") != "" {
		app.setCalls(f["call"])
	}
	if f.Get("expires") != "" {
		err := app.setExpires(f.Get("expires"))
		if err != nil {
			w.Write(wx.JsonResponse(err))
			return
		}
	}

	// store app info
	err = NewStorage().SaveApp(app)
	if err != nil {
		w.Write(wx.JsonResponse(err))
		return
	}
	w.Write(wx.JsonResponse(nil))
	return
}

func (srv *RegisterServer) checkPrivilage(key, appid, secret string) (wxErr wx.WxError) {
	return
}
