package wechat

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiServer(t *testing.T) {
	appid := "wx06766a90ab72960e"
	secret := "05bd8b6064a9941b72ee44d5b3bfdb6a"

	ts_data := []struct {
		url    string
		fields []string
	}{
		{
			url:    fmt.Sprintf("/api?appid=%s&secret=%s", appid, secret),
			fields: []string{"access_token", "expires_in"},
		},
		{
			url:    fmt.Sprintf("/api?appid=%s&secret=%s", appid+"xxx", secret),
			fields: []string{"errcode", "errmsg"},
		},
		{
			url:    fmt.Sprintf("/api?appid=%s&secret=%s", appid, secret+"xxx"),
			fields: []string{"errcode", "errmsg"},
		},
	}

	srv := NewApiServer()
	ts := httptest.NewServer(srv)
	defer ts.Close()

	for _, v := range ts_data {
		resp, err := http.Get(ts.URL + v.url)
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		var f interface{}
		err = json.Unmarshal(body, &f)
		if err != nil {
			log.Fatal(err)
		}

		m := f.(map[string]interface{})
		for _, fld := range v.fields {
			if m[fld] == nil {
				log.Fatal()
			}
		}
	}
}
