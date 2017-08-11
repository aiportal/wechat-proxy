// Copyright 2017 https://github.com/aiportal.
// All rights reserved.

package wxproxy

import (
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
)

// wechat error response
type wxError struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

func (e *wxError) Success() bool {
	return e.ErrCode == 0
}

func (e *wxError) String() string {
	return fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
}

func (e *wxError) Serialize() []byte {
	js := fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
	return []byte(js)
}

func newError(err error) *wxError {
	e := new(wxError)
	e.ErrCode = -10001
	e.ErrMsg = err.Error()
	return e
}

func httpGetJson(url string, obj interface{}) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, obj)
	return
}
