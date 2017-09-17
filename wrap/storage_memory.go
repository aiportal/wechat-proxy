// +build !sqlite

package wrap

import (
	"time"
	wx "wechat-proxy/wechat"
	"fmt"
	"errors"
)

const (
	storeCacheDuration = 365 * 24 * time.Hour
	storeCacheLimit    = 1000
)

var storage *Storage

type Storage struct {
	appMap *wx.CacheMap
	userMap *wx.CacheMap
}

func NewStorage() *Storage {
	if storage == nil {
		s := new(Storage)
		s.appMap = wx.NewCacheMap(storeCacheDuration, storeCacheLimit)
		s.userMap = wx.NewCacheMap(storeCacheDuration, storeCacheLimit)
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
		err = ErrNotFound
		return
	}
	r := v.(WxApp)
	if r.isExpired() {
		return
	}
	app = &r
	return
}

func (s *Storage) SaveUser(user *WxUser) (err error) {
	key := fmt.Sprintf("%s-%s", user.AppId, user.OpenId)
	s.userMap.Set(key, *user)
	return
}

func (s *Storage) LoadUser(appid, openid string) (user *WxUser, err error) {
	key := fmt.Sprintf("%s-%s", appid, openid)
	v, ok := s.userMap.Get(key)
	if !ok {
		err = ErrNotFound
		return
	}
	r := v.(WxUser)
	user = &r
	return
}

var ErrNotFound = errors.New("not found")
