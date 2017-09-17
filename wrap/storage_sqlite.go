// +build sqlite

package wrap

import (
	"errors"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"time"
)

const (
	APP_DB_TYPE = "sqlite3"
	APP_DB_NAME = "wxproxy.db"
)

type Storage struct {
}

func NewStorage() *Storage {
	return new(Storage)
}

func (*Storage) db(f func(*gorm.DB)) {
	db, err := gorm.Open(APP_DB_TYPE, APP_DB_NAME)
	if err != nil {
		return
	}
	defer db.Close()
	f(db)
}

func (*Storage) SaveApp(app *WxApp) (err error) {
	db, err := gorm.Open(APP_DB_TYPE, APP_DB_NAME)
	if err != nil {
		return
	}
	defer db.Close()
	db.AutoMigrate(&WxApp{})

	err = db.Save(app).Error
	db.Where("Expires < ?", time.Now()).Delete(WxApp{})
	return
}

func (*Storage) LoadApp(key string) (app *WxApp, err error) {
	db, err := gorm.Open(APP_DB_TYPE, APP_DB_NAME)
	if err != nil {
		return
	}
	defer db.Close()

	r := WxApp{}
	err = db.Where("Key = ?", key).First(&r).Error
	if err != nil {
		return
	}
	if r.isExpired() {
		return
	}

	app = &r
	return
}

func (s *Storage) SaveUser(user *WxUser) (err error) {
	s.db(func(db *gorm.DB) {
		db.AutoMigrate(&WxUser{})
		err = db.Save(user).Error
	})
	return
}

func (s *Storage) LoadUser(appid, openid string) (user *WxUser, err error) {
	s.db(func(db *gorm.DB) {
		r := WxUser{}
		err = db.Where("appid = ? AND openid = ?", appid, openid).First(&r).Error
		user = &r
	})
	return
}

var ErrNotFound = errors.New("not found")
