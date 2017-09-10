// +build !sqlite

package wrap

import (
	"time"
	wx "wechat-proxy/wechat"
)

const (
	storeCacheDuration = 365 * 24 * time.Hour
	storeCacheLimit    = 1000
)

var storage *Storage

type Storage struct {
	appMap *wx.CacheMap
}

func NewStorage() *Storage {
	if storage == nil {
		s := new(Storage)
		s.appMap = wx.NewCacheMap(storeCacheDuration, storeCacheLimit)
		storage = s
	}
	return storage
}

func (s *Storage) SaveApp(app *WxApp) (err error) {
	s.appMap.Set(app.Key, *app)
	s.appMap.Shrink()
	return
}

func (s *Storage) LoadApp(key string) (app *WxApp, err error) {
	v, ok := s.appMap.Get(key)
	if !ok {
		err = wx.ErrCacheTimeout
		return
	}
	r := v.(WxApp)
	if r.isExpired() {
		return
	}
	app = &r
	return
}
