package wxproxy

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sort"
	"strings"
	"time"
)

type wxCryptoMsg struct {
	ToUserName string
	Encrypt    string
}

type wxCryptoReply struct {
	XMLName      xml.Name `xml:"xml"`
	Encrypt      CDATA
	MsgSignature CDATA
	TimeStamp    int64
	Nonce        CDATA
}

type wechatMsgCrypter struct {
	Token  string
	AesKey []byte
}

func NewCrypter(token, aes_key string) (mc *wechatMsgCrypter, err error) {
	mc = new(wechatMsgCrypter)
	mc.Token = token
	mc.AesKey, err = base64.StdEncoding.DecodeString(aes_key + "=")
	return
}

func (*wechatMsgCrypter) sha1Signature(args ...string) string {
	sort.Strings(args)
	s := strings.Join(args, "")
	hash := sha1.Sum([]byte(s))
	return fmt.Sprintf("%x", hash)
}

// extract message string from encrypt package
func (mc *wechatMsgCrypter) decryptMsg(body io.Reader) (msg []byte, appid string, err error) {
	body_bytes, err := ioutil.ReadAll(body)
	if err != nil {
		log.Println(err.Error())
		return
	}

	var pkg wxCryptoMsg
	err = xml.Unmarshal(body_bytes, &pkg)
	if err != nil {
		return
	}

	data, err := base64.StdEncoding.DecodeString(pkg.Encrypt)
	if err != nil {
		return
	}
	msg, appid, err = mc.decryptMsgBody(data)
	return
}

// generate reply package
func (mc *wechatMsgCrypter) encryptMsg(msg []byte, appid string) (body []byte, err error) {
	data, err := mc.encryptMsgBody(msg, appid)
	if err != nil {
		return
	}
	encrypt := base64.StdEncoding.EncodeToString(data)

	timestamp := time.Now().Unix()
	nonce := randomString(16)
	sign := mc.sha1Signature(mc.Token, fmt.Sprintf("%d", timestamp), nonce, encrypt)

	pkg := wxCryptoReply{
		Encrypt:   CDATA(encrypt),
		TimeStamp: timestamp,
		Nonce:     CDATA(nonce),
	}
	pkg.MsgSignature = CDATA(sign)
	body, err = xml.Marshal(pkg)
	return
}

func (mc *wechatMsgCrypter) decryptMsgBody(data []byte) (msg []byte, appid string, err error) {
	c, err := aes.NewCipher(mc.AesKey)
	if err != nil {
		return
	}
	cbc := cipher.NewCBCDecrypter(c, mc.AesKey[:16])
	cbc.CryptBlocks(data, data)
	data = mc.decodePKCS7(data)

	// get length of xml text
	// [0:16]: random
	// [16:20]: length
	// [20:len+20]: xml
	// [len+20:]: appid
	var msg_len int32
	buf := bytes.NewBuffer(data[16:20])
	binary.Read(buf, binary.BigEndian, &msg_len)

	msg = data[20 : 20+msg_len]
	appid = string(data[20+msg_len:])
	return
}

func (mc *wechatMsgCrypter) encryptMsgBody(msg []byte, appid string) (data []byte, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, int32(len(msg)))
	if err != nil {
		return
	}
	msg_len := buf.Bytes()

	rand_bytes := make([]byte, 16)
	_, err = io.ReadFull(rand.Reader, rand_bytes)
	if err != nil {
		return
	}
	msg_bytes := bytes.Join([][]byte{rand_bytes, msg_len, msg, []byte(appid)}, nil)
	msg_bytes = mc.encodePKCS7(msg_bytes)

	c, err := aes.NewCipher(mc.AesKey)
	if err != nil {
		return
	}
	cbc := cipher.NewCBCEncrypter(c, mc.AesKey[:16])
	cbc.CryptBlocks(msg_bytes, msg_bytes)

	data = msg_bytes
	return
}

func (*wechatMsgCrypter) decodePKCS7(text []byte) []byte {
	pad := int(text[len(text)-1])
	if pad < 1 || pad > 32 {
		pad = 0
	}
	return text[:len(text)-pad]
}

func (*wechatMsgCrypter) encodePKCS7(text []byte) []byte {
	const BlockSize = 32
	amount := BlockSize - len(text)%BlockSize
	for i := 0; i < amount; i++ {
		text = append(text, byte(amount))
	}
	return text
}
