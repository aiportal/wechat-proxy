package enterprise

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQYApiServer(t *testing.T) {
	appid := "wx2c67ebb55a4012c3"
	secret := "jVgdNTMwvUw0QpEnp3XCCuntS22gM5JT50FmvKtL-F8"

	ts_data := []struct {
		url    string
		fields []string
	}{
		{
			url:    fmt.Sprintf("/qyapi?appid=%s&secret=%s", appid, secret),
			fields: []string{"access_token", "expires_in"},
		},
		{
			url:    fmt.Sprintf("/qyapi?appid=%s&secret=%s", appid+"xxx", secret),
			fields: []string{"errcode", "errmsg"},
		},
		{
			url:    fmt.Sprintf("/qyapi?appid=%s&secret=%s", appid, secret+"xxx"),
			fields: []string{"errcode", "errmsg"},
		},
	}

	ts := httptest.NewServer(NewQyServer())
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
