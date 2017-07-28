package wxproxy

import (
	"testing"
	"net/http/httptest"
	"net/http"
	"log"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"net/url"
)

func TestAuthInit(t *testing.T) {

	ts_init := []struct{
		Url string
		Redirect string
		Fields []string
	}{
		{
			Url: "/auth?appid=wx000&secret=xxxxxx",
			Redirect: "/echo",
			Fields:[]string{"auth_uri", "expires_in"},
		},
		{
			Url: "/auth?appid=wx000",
			Redirect: "/echo",
			Fields:[]string{"errcode", "errmsg"},
		},
	}

	ts := httptest.NewServer(NewAuthServer())
	defer ts.Close()

	for _, v := range ts_init {
		_url := fmt.Sprintf("%s%s&redirect_uri=%s", ts.URL, v.Url, url.PathEscape(ts.URL+v.Redirect))
		resp, err := http.Get(_url)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(_url)

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Printf("body: %s\n", string(body))

		var f interface{}
		err = json.Unmarshal(body, &f)
		if err != nil {
			fmt.Printf("url: %s\n", _url)
			fmt.Printf("body: %s\n", string(body))
			log.Fatal(err)
		}

		m := f.(map[string]interface{})
		for _, fld := range v.Fields {
			if m[fld] == nil {
				log.Fatal()
			}
		}
	}
}
