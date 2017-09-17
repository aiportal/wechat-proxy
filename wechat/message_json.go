package wechat

import (
	"reflect"
)

var wxEventsMap = map[string]reflect.Type{
	"subscribe":   nil,
	"unsubscribe": nil,
	"SCAN":        nil,
	"LOCATION":    nil,
	"CLICK":       nil,
	"VIEW":        nil,
}

func init() {
	event_maps := []struct {
		Names []string
		Type  reflect.Type
	}{
		{
			Names: []string{
				"card_pass_check",
				"card_not_pass_check",
				"user_get_card",
				"user_gifting_card",
				"user_del_card",
				"user_consume_card",
				"user_pay_from_pay_cell",
				"user_view_card",
				"user_enter_session_from_card",
				"update_member_card",
				"card_sku_remind",
				"card_pay_order",
				"submit_membercard_user_info",
			},
			Type: reflect.TypeOf(wxEventCard{}),
		},
		{
			Names: []string{
				"user_scan_product",
				"user_scan_product_enter_session",
				"user_scan_product_async",
				"user_scan_product_verify_action",
			},
			Type: reflect.TypeOf(wxEventPruduct{}),
		},
		{
			Names: []string{
				"qualification_verify_success",
				"qualification_verify_fail",
				"naming_verify_success",
				"naming_verify_fail",
				"annual_renew",
				"verify_expired",
			},
			Type: reflect.TypeOf(wxEventVerify{}),
		},
	}
	wxEventsMap["ShakearoundUserShake"] = reflect.TypeOf(wxEventBeacon{})
	wxEventsMap["WifiConnected"] = reflect.TypeOf(wxEventWifi{})

	for _, v := range event_maps {
		for _, name := range v.Names {
			wxEventsMap[name] = v.Type
		}
	}
}

// xml to json
// cover 90% wechat message and events
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140453
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140454
type WxMessage struct {
	ToUserName   string
	FromUserName string
	CreateTime   uint64
	MsgType      string

	MsgId        uint64  `json:",omitempty"` // message
	Content      string  `json:",omitempty"` // message: text
	MediaId      string  `json:",omitempty"` // message: picture,voice,video
	PicUrl       string  `json:",omitempty"` // message: picture
	Format       string  `json:",omitempty"` // message: voice
	Recognition  string  `json:",omitempty"` // message: voice
	ThumbMediaId string  `json:",omitempty"` // message: video
	Location_X   float64 `json:",omitempty"` // message: geometry
	Location_Y   float64 `json:",omitempty"` // message: geometry
	Scale        int32   `json:",omitempty"` // message: geometry
	Label        string  `json:",omitempty"` // message: geometry
	Title        string  `json:",omitempty"` // message: link
	Description  string  `json:",omitempty"` // message: link
	Url          string  `json:",omitempty"` // message: link

	Event     string  `json:",omitempty"` // event
	EventKey  string  `json:",omitempty"` // event: menu,scan
	Ticket    string  `json:",omitempty"` // event: scan
	Latitude  float64 `json:",omitempty"` // event: location
	Longitude float64 `json:",omitempty"` // event: location
	Precision float64 `json:",omitempty"` // event: location
}

type wxEvent struct {
	ToUserName   CDATA
	FromUserName CDATA
	CreateTime   uint64
	MsgType      CDATA
	Event        CDATA
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1451025274
type wxEventCard struct {
	wxEvent
	CardId              CDATA  `json:",omitempty"` // user_del_card, user_enter_session_from_card, submit_membercard_user_info
	UserCardCode        CDATA  `json:",omitempty"`
	OuterStr            CDATA  `json:",omitempty"` // user_view_card
	RefuseReason        CDATA  `json:",omitempty"` // card_pass_check, card_not_pass_check
	IsGiveByFriend      uint32 `json:",omitempty"` // user_get_card
	FriendUserName      CDATA  `json:",omitempty"`
	OldUserCardCode     CDATA  `json:",omitempty"`
	OuterId             uint32 `json:",omitempty"`
	IsRestoreMemberCard uint32 `json:",omitempty"`
	IsRecommendByFriend uint32 `json:",omitempty"`
	IsReturnBack        uint32 `json:",omitempty"` // user_gifting_card
	IsChatRoom          uint32 `json:",omitempty"`
	ConsumeSource       CDATA  `json:",omitempty"` // user_consume_card
	LocationName        CDATA  `json:",omitempty"`
	StaffOpenId         CDATA  `json:",omitempty"`
	VerifyCode          CDATA  `json:",omitempty"`
	RemarkAmount        CDATA  `json:",omitempty"`
	TransId             CDATA  `json:",omitempty"` // user_pay_from_pay_cell
	LocationId          uint64 `json:",omitempty"`
	Fee                 CDATA  `json:",omitempty"`
	OriginalFee         CDATA  `json:",omitempty"`
	ModifyBonus         int32  `json:",omitempty"` // update_member_card
	ModifyBalance       int32  `json:",omitempty"`
	Detail              CDATA  `json:",omitempty"` // card_sku_remind
	OrderId             CDATA  `json:",omitempty"` // card_pay_order
	Status              CDATA  `json:",omitempty"`
	CreateOrderTime     uint64 `json:",omitempty"`
	PayFinishTime       uint64 `json:",omitempty"`
	Desc                CDATA  `json:",omitempty"`
	FreeCoinCount       CDATA  `json:",omitempty"`
	PayCoinCount        CDATA  `json:",omitempty"`
	RefundFreeCoinCount CDATA  `json:",omitempty"`
	RefundPayCoinCount  CDATA  `json:",omitempty"`
	OrderType           CDATA  `json:",omitempty"`
	Memo                CDATA  `json:",omitempty"`
	ReceiptInfo         CDATA  `json:",omitempty"`
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1455872179
type wxEventPruduct struct {
	wxEvent
	KeyStandard string `json:",omitempty"`
	KeyStr      string `json:",omitempty"`
	Country     string `json:",omitempty"`
	Province    string `json:",omitempty"`
	City        string `json:",omitempty"`
	Sex         uint32 `json:",omitempty"`
	Scene       uint32 `json:",omitempty"`
	ExtInfo     string `json:",omitempty"`
	RegionCode  string `json:",omitempty"`
	Result      string `json:",omitempty"`
	ReasonMsg   string `json:",omitempty"`
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1443448066
type wxEventBeacon struct {
	wxEvent
	ChosenBeacon *struct {
		Uuid     string
		Major    string
		Minor    string
		Distance float32
	}
	AroundBeacons []struct {
		AroundBeacon struct {
			Uuid     string
			Major    string
			Minor    string
			Distance float32
		}
	}
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1455785130
type wxEventVerify struct {
	wxEvent
	ExpiredTime uint64 `json:",omitempty"`
	FailTime    uint64 `json:",omitempty"`
	FailReason  string `json:",omitempty"`
}

// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1444894131
type wxEventWifi struct {
	wxEvent
	ConnectTime uint64
	ExpireTime  uint64
	VendorId    string
	ShopId      string
	DeviceNo    string
}

// json to xml
// https://mp.weixin.qq.com/wiki?t=resource/res_main&id=mp1421140543
type WxReply struct {
	ToUserName   string
	FromUserName string
	CreateTime   uint64
	MsgType      string

	// text
	Content CDATA `xml:",omitempty"`

	// image
	Image *struct {
		MediaId CDATA
	} `xml:",omitempty"`

	// voice
	Voice *struct {
		MediaId CDATA
	} `xml:",omitempty"`

	// video
	Video *struct {
		MediaId     CDATA
		Title       CDATA
		Description CDATA
	} `xml:",omitempty"`

	// music
	Music *struct {
		Title        CDATA
		Description  CDATA
		MusicUrl     CDATA
		HQMusicUrl   CDATA
		ThumbMediaId CDATA
	} `xml:",omitempty"`

	// news
	ArticleCount int32 `xml:",omitempty"`
	Articles     []struct {
		Item struct {
			Title       CDATA
			Description CDATA
			PicUrl      CDATA
			Url         CDATA
		}
	} `xml:",omitempty"`
}
