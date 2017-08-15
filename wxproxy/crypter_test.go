package wxproxy

import (
	"testing"
	"bytes"
	"strings"
)

func TestDecrypt(t *testing.T) {

	ts_data := []struct{
		AppId string
		Token string
		AesKey string
		Url string
		Package string
		Encrypt string
	}{
		{
			AppId: "wx06766a90ab72960e",
			Token: "www.aiportal.net",
			AesKey: "XVeChLv7XLCpkHiPJTGrx6Ha18Yq9i6LCkHV1oxk3mw",
			Url: "/crypto?&signature=5c5d814245855eb485df751e7b9be4a7f2622133&timestamp=1502757162&nonce=1106965505&encrypt_type=aes&msg_signature=29e271bff094d2acbda57c07ad6c974042e0128c",
			Package: `<xml>
<ToUserName><![CDATA[bfbd]]></ToUserName>
<Encrypt><![CDATA[ajTxGtmDjoECHpJkotp9Ok8elXjtUQ/BP1F795qu/7r9Efmeni7sRXS7f/RfJNgTshi/8XbiKx72Nri3kltaJX1t3QpmUvNufD7dA3ekwVp/1DLcGP65YtSgsrBTa9RoVEvby23X+7+X4mhBM5JzS8YFztsJEw3vxF5iYFOV4rdFrszli1ddaZRNZAGQDabcJ/rQIONxcog0t5ZGUIb+HuawqpNGtfE/wOmMJ0P5KVrkZP9U2+RbMCJQS8+HPUxs7ofJL7E7KicJ3JS41fDXI2IJjVTGOO+ddBmQVXLPX0xvKVUtjxj0VPea8/lFKSUIQlnqzWxJ7QP9/XpYIWVWHhNr2O3fXQ5SfberZlEPCEuudHklsjyOueDet06rNF5+28v0TIGuT7OdjolTG6r/oSyMRlO+DsKiyIaWWn0a8e8Y+CO8F4hoPdM9NWHW4pxCeKatu7nsAQOfWZc3pcnHBo8+60TLnmDfmR6eSgQTnbPmetgrDsxFtOgebk+y4nLq]]></Encrypt>
</xml>`,
		},
		{
			AppId: "wx06766a90ab72960e",
			Token: "www.aiportal.net",
			AesKey: "XVeChLv7XLCpkHiPJTGrx6Ha18Yq9i6LCkHV1oxk3mw",
			Url: "/crypto?signature=9634db1ea2c276e10dafc29a8e781a19892535c5&timestamp=1502758613&nonce=951682770&encrypt_type=aes&msg_signature=d3f6647cb1f160aee9a8826133ff731883889a76",
			Package: `<xml>
<ToUserName><![CDATA[bfbd]]></ToUserName>
<Encrypt><![CDATA[l63nDQa0ch+gmwilKdIGBi74a5ttz6mBwgjq3Pvsq7lU90nFYitSkIbxLNQdEqLx6lXHUrDsycT2V3M1xcbEoAr1aLFKzYlFJeM0Qp9UYNTNd8E6Qk5Wnz42yxkCr6ZO1WIBzlnLOahD0rGN+kx1s/EsrUHPu7SrgO9AQgsZgv8Y4wHWwbdtLLjr+e9T+5Gf93k0w1JJrzMlQfMbx9PlHaUdPcvHIgqPeCt0b92bQfciSUDlLygo+8iFCJ/Q2fPtWOKSFOKs42sAgPQn7IaN/gzmPKuWjT/NSxxBycRFSl+fQZfca63OxawyleF5SRcgnhjxBPb+VCCTqRGh0AoIjVYP7rkbW2Hk0U5qCBsW8VufB9jGScxTwHt4njqDqVe7/mKi28T92lwCCjGwPf7zpyaKOH9M1e7kNzmYXPQMhLcQB86sOuLDSerUUdBCHuYc5iRTMfF+Q7CYEFk2YeOKkgggD1ZaAdcpyEENITiMJIor4RoNVbGJIPXRCF57YNf9SpMThEs4JWGov8dFiCTsogwoY7jww77pP2dsafoATBo=]]></Encrypt>
</xml>`,
		},
	}

	for _, v := range ts_data {
		c, err := NewCrypter(v.Token, v.AesKey)
		if err != nil {
			t.Fatal(err)
		}
		msg, appid, err := c.decryptMsg(bytes.NewReader([]byte(v.Package)))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(msg), "<xml>") {
			t.Fatal("decrypt fail")
		}
		if appid != v.AppId {
			t.Fatal("decrypt appid error")
		}
	}
}

func TestEncrypt(t *testing.T) {
	ts_data := []struct{
		AppId string
		Token string
		AesKey string
		Message string
	}{
		{
			AppId: "wx06766a90ab72960e",
			Token: "www.aiportal.net",
			AesKey: "XVeChLv7XLCpkHiPJTGrx6Ha18Yq9i6LCkHV1oxk3mw",
			Message: `<xml>
<ToUserName><![CDATA[toUser]]></ToUserName>
<FromUserName><![CDATA[fromUser]]></FromUserName>
<CreateTime>12345678</CreateTime>
<MsgType><![CDATA[text]]></MsgType>
<Content><![CDATA[你好]]></Content>
</xml>`,
		},
	}
	for _, v := range ts_data {
		c, err := NewCrypter(v.Token, v.AesKey)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := c.encryptMsg([]byte(v.Message), v.AppId)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(pkg), "<xml>") {
			t.Fatal("encrypt fail")
		}

		msg, appid, err := c.decryptMsg(bytes.NewReader([]byte(pkg)))
		if string(msg) != v.Message {
			t.Fatal("decrypt fail")
		}
		if appid != v.AppId {
			t.Fatal("decrypt appid error")
		}
	}
}