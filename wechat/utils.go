// Copyright 2017 https://github.com/aiportal.
// All rights reserved.

package wechat

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

// wechat error response
type WxError struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

func (e *WxError) Success() bool {
	return e.ErrCode == 0
}

func (e *WxError) String() string {
	return fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
}

func (e *WxError) Serialize() []byte {
	js := fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
	return []byte(js)
}

func NewErrorStr(msg string) *WxError {
	e := new(WxError)
	e.ErrCode = -10001
	e.ErrMsg = msg
	return e
}

func NewError(err error) *WxError {
	e := new(WxError)
	e.ErrCode = -10001
	e.ErrMsg = err.Error()
	return e
}

func HttpGetJson(url string, obj interface{}) (body []byte, err error) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	if obj != nil {
		err = json.Unmarshal(body, obj)
	}
	return
}

// create a json formatted response
func JsonResponse(obj interface{}) []byte {
	if obj == nil {
		return []byte(`{"success":true}`)
	}
	switch v := obj.(type) {
	case error:
		return NewError(v).Serialize()
	default:
		bs, err := json.Marshal(obj)
		if err != nil {
			return NewError(err).Serialize()
		}
		return bs
	}
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	rand.Seed(time.Now().Unix())
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

type CDATA string

func (c CDATA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(struct {
		string `xml:",cdata"`
	}{string(c)}, start)
}
