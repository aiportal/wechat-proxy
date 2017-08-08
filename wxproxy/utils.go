// Copyright 2017 https://github.com/aiportal.
// All rights reserved.

package wxproxy

import (
	"fmt"
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
	if e.ErrCode == 0 {
		return ""
	} else {
		return fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
	}
}

func newError(err error) *wxError {
	e := new(wxError)
	e.ErrCode = -10001
	e.ErrMsg = err.Error()
	return e
}

func (e *wxError) Error() string {
	if e.ErrCode == 0 {
		return ""
	}
	js := fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
	return js
}
