package wxproxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestQYApiServer(t *testing.T) {
	appid := os.Args[4]
	secret := os.Args[5]

	ts_data := []struct {
		url    string
		fields []string
	}{
		{
			url:    fmt.Sprintf("/qyapi?corpid=%s&corpsecret=%s", appid, secret),
			fields: []string{"access_token", "expires_in"},
		},
		{
			url:    fmt.Sprintf("/qyapi?corpid=%s&corpsecret=%s", appid + "xxx", secret),
			fields: []string{"errcode", "errmsg"},
		},
	}

	ts := httptest.NewServer(NewQYApiServer())
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
