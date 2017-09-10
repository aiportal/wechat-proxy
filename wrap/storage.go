package wrap

import (
	"strconv"
	"strings"
	"time"
)

type WxApp struct {
	Key       string     `gorm:"not null; primary_key;"`
	AppId     string     `gorm:"not null; index;"`
	Secret    string     `gorm:"not null;"`
	Token     string     // 消息加解密Token
	AesKey    string     // 消息加解密秘钥
	MchId     string     // 微信支付账号
	MchKey    string     // 微信支付秘钥
	IpAddress string     // 服务器IP地址(微信支付)
	Calls     string     // 允许调用的接口列表，NULL表示不限制
	Expires   *time.Time // 过期时间，NULL表示永久
}

func (app *WxApp) setExpires(expires string) (err error) {
	if expires == "" {
		app.Expires = nil
		return
	}
	seconds, err := strconv.Atoi(expires)
	if err != nil {
		return
	}
	tm := time.Now().Add(time.Duration(seconds) * time.Second)
	app.Expires = &tm
	return
}

func (app *WxApp) isExpired() bool {
	if app.Expires != nil {
		if app.Expires.Before(time.Now()) {
			return true
		}
	}
	return false
}

func (app *WxApp) setCalls(calls []string) {
	app.Calls = strings.Join(calls, "|")
}

func (app *WxApp) inCalls(path string) bool {
	if app.Calls == "" {
		return true
	}
	calls := strings.Split(app.Calls, "|")
	for _, v := range calls {
		if strings.HasPrefix(path, v) {
			return true
		}
	}
	return false
}
