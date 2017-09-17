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

func (app *WxApp) setExpires(str string) (err error) {
	app.Expires = nil
	if str == "" {
		return
	}
	seconds, err := strconv.Atoi(str)
	if err != nil {
		return
	}
	tm := time.Now().Add(time.Duration(seconds) * time.Second)
	app.Expires = &tm
	return
}

func (app *WxApp) isExpired() bool {
	if app.Expires != nil {
		return app.Expires.Before(time.Now())
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

type WxSubscribe struct {
	AppId           string     `gorm:"column:appid; not null; primary_key"`  // 公众号的APPID
	OpenId          string     `gorm:"column:openid; not null; primary_key"` // 用户的标识，对当前公众号唯一
	UnionId         string     `gorm:"column:unionid; not null; index"`      // 只有在用户将公众号绑定到微信开放平台帐号后，才会出现该字段。
	Subscribe       bool       `gorm:"column:subscribe"`                     // 用户是否订阅该公众号标识，值为0时，代表此用户没有关注该公众号，拉取不到其余信息。
	SubscribeTime   *time.Time `gorm:"column:subscribe_time"`                // 用户关注时间，为时间戳。如果用户曾多次关注，则取最后关注时间
	UnSubscribeTime *time.Time `gorm:"column:unsubscribe_time"`              // 用户取消关注时间，为时间戳。
	Referral        string     `gorm:"column:Referral"`                      // 推荐人
}

type WxUser struct {
	AppId           string `gorm:"column:appid; not null; primary_key"`  // 公众号的APPID
	OpenId          string `gorm:"column:openid; not null; primary_key"` // 用户的标识，对当前公众号唯一
	UnionId         string `gorm:"column:unionid; not null; index"`      // 只有在用户将公众号绑定到微信开放平台帐号后，才会出现该字段。
	Subscribe       bool   `gorm:"column:subscribe"`                     // 用户是否订阅该公众号标识，值为0时，代表此用户没有关注该公众号，拉取不到其余信息。
	SubscribeTime   uint64 `gorm:"column:subscribe_time"`                // 用户关注时间，为时间戳。如果用户曾多次关注，则取最后关注时间
	UnSubscribeTime uint64 `gorm:"column:unsubscribe_time"`              // 用户取消关注时间，为时间戳。
	Referral        string `gorm:"column:Referral"`                      // 推荐人

	Nickname   string `gorm:"column:nickname; not null; index"`      // 用户的昵称
	Sex        int `gorm:"column:sex"`                            // 用户的性别，值为1时是男性，值为2时是女性，值为0时是未知
	City       string `gorm:"column:city"`                           // 用户所在城市
	Country    string `gorm:"column:country"`                        // 用户所在国家
	Province   string `gorm:"column:province"`                       // 用户所在省份
	Language   string `gorm:"column:language"`                       // 用户的语言，简体中文为zh_CN
	HeadImgUrl string `gorm:"column:headimgurl; type:varchar(2000)"` // 用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），用户没有头像时该项为空。若用户更换头像，原有头像URL将失效。
	Remark     string `gorm:"column:remark; type:varchar(2000)"`     // 公众号运营者对粉丝的备注，公众号运营者可在微信公众平台用户管理界面对粉丝添加备注

	Group string `gorm:"column:group"` // 用户所在的分组（兼容旧的用户分组接口）
	Tags  string `gorm:"column:tags"`  // 用户被打上的标签

	Latitude     float64 `gorm:"column:latitude"`      // 地理位置纬度
	Longitude    float64 `gorm:"column:longitude"`     // 地理位置经度
	Precision    float64 `gorm:"column:precision"`     // 地理位置精度
	LocationTime uint64  `gorm:"column:location_time"` // 最后定位时间
}
