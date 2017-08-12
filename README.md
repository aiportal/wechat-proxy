# wechat-proxy  
A proxy server for wechat access_token, callback messages, oauth api and jsapi.  

Auto cache access_token, jsapi ticket, cart_ticket until expires.   
Dispatch callback messages to multiple server.  
Simple oauth process for wechat browser and web page.

Access token can be shared by multiple process or multiple server.  

    https://wx.aiportal.net/api?appid=APPID&secret=SECRET
    https://wx.aiportal.net/qyapi?corpid=CORPID&corpsecret=SECRET

Jsapi ticket and card ticket can be shared by multiple process or multiple server.
      
    https://wx.aiportal.net/js?corpid=CORPID&corpsecret=SECRET
    https://wx.aiportal.net/card?corpid=CORPID&corpsecret=SECRET

Callback message can dispatch to multi server.  
If multiple call address has been set, first none empty result will be return to wechat server.

    https://wx.aiportal.net/msg?call=https%3A//weixin.ultragis.com&call=www.ultragis.com  

Wechat oauth simple process:

    https://wx.aiportal.net/auth?appid=APPID&secret=SECRET&redirect_uri=URL&scope=snsapi_userinfo&state=random
    {"auth_uri":"http://wx.aiportal.net/auth?authid=8DBB11A42759FDAF95C9C9005E30CC86", "expires_in":300}

Send auth_uri to wechat browser or scan qrcode by wechat browser, redirect_uri will receive user_info directly.


## 微信代理  
用于全局缓存微信 access_token, ticket, 多路转发微信回调消息以及简化微信 oauth2 认证的服务器程序。 
 
自动缓存微信的 access_token, jsapi ticket, card ticket.
将微信回调消息转发至多个后台服务器。
简化微信 oath2 认证流程，支持Web页面扫码登录和手机端直接登录。

access_token 全局缓存:

    https://wx.aiportal.net/api?appid=APPID&secret=SECRET
    https://wx.aiportal.net/qyapi?corpid=CORPID&corpsecret=SECRET

JSSDK ticket 和 卡券 ticket 全局缓存:
      
    https://wx.aiportal.net/js?corpid=CORPID&corpsecret=SECRET
    https://wx.aiportal.net/card?corpid=CORPID&corpsecret=SECRET

微信回调消息的多路转发：  
如果设置了多个 call 参数，系统会将第一个非空的请求结果返回给微信服务器。  <br/>

    https://wx.aiportal.net/msg?call=https%3A//weixin.ultragis.com&call=www.ultragis.com  

微信登录认证:

    https://wx.aiportal.net/auth?appid=APPID&secret=SECRET&redirect_uri=URL&scope=snsapi_userinfo&state=random
    {"auth_uri":"http://wx.aiportal.net/auth?authid=8DBB11A42759FDAF95C9C9005E30CC86", "expires_in":300}

只需将 auth_uri 发送至微信浏览器或使用微信浏览器扫码，redirect_uri 即可获得用户信息(post json)。
