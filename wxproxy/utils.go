// Copyright 2017 https://github.com/aiportal.
// All rights reserved.

package wxproxy

import (
	"fmt"
	"time"
)

// access_token stay time in memory
var TokenCacheDuration = 3600 * time.Second

// access_token max count in memory
var TokenCacheLimit = 100

// wechat error response
type wxError struct {
	ErrCode int    `json:"errcode,omitempty"`
	ErrMsg  string `json:"errmsg,omitempty"`
}

func (e *wxError) Success() bool {
	return e.ErrCode == 0
}

func (e *wxError) Error() string {
	if e.ErrCode == 0 {
		return ""
	}
	js := fmt.Sprintf(`{"errcode": %d, "errmsg": "%s"}`, e.ErrCode, e.ErrMsg)
	return js
}
