package wxproxy

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"
)

// access_token stay time in memory
var TokenCacheDuration = 3600 * time.Second

// access_token max count in memory
var TokenCacheLimit = 100

type wxError struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func parseError(data []byte) (err error) {
	var e wxError
	err = json.Unmarshal(data, &e)
	if err != nil {
		return
	}
	if e.ErrCode != 0 {
		err = errors.New(string(data))
	}
	return
}

func httpGet(url string) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	return
}

//type wxAccessToken struct {
//	wxError
//
//	AccessToken string  `json:"access_token"`
//	ExpiresIn int       `json:"expires_in"`
//}
