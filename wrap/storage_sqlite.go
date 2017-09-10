// +build sqlite

package wrap

import (
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
