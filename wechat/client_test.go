package wechat

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWechatClient(t *testing.T) {

	ts_data := []struct {
		Appid  string
		Secret string
	}{
		{
			Appid:  "wx06766a90ab72960e",
			Secret: "05bd8b6064a9941b72ee44d5b3bfdb6a",
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/api", NewApiServer())
	mux.Handle("/jsapi", NewJsTicketServer())
	ts := httptest.NewServer(mux)
	defer ts.Close()

	for _, v := range ts_data {
		c := &WechatClient{}

		// test access_token
		token1, err := c.GetAccessToken(ts.URL, v.Appid, v.Secret)
		if err != nil {
			t.Fatal(err)
			return
		}
		if token1 == "" {
			t.Fatal("empty access_token")
			return
		}
		token2, err := c.GetAccessToken(ts.URL, v.Appid, v.Secret)
		if err != nil {
			t.Fatal(err)
			return
		}
		if token2 != token1 {
			t.Fatal("access_token cache error")
			return
		}

		// test js ticket
		ticket1, err := c.GetJsTicket(ts.URL, v.Appid, v.Secret)
		if err != nil {
			t.Fatal(err)
			return
		}
		if ticket1 == "" {
			t.Fatal("empty jsapi_ticket")
			return
		}
		ticket2, err := c.GetJsTicket(ts.URL, v.Appid, v.Secret)
		if err != nil {
			t.Fatal(err)
			return
		}
		if ticket2 != ticket1 {
			t.Fatal("jsapi_ticket cache error")
			return
		}
	}
}
