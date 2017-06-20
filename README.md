# wechat-proxy  
A proxy server for wechat access_token and callback messages.  
Auto cache access_token until expires and dispatch callback messages to multiple server.  
  
Simple replace:  
    https://api.weixin.<b>qq</b>.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET  
to:  
    https://api.weixin.<b>ultragis</b>.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET  
and access_token can be shared by multiple process or multiple server.  
  
Simple replace:  
    https://qyapi.weixin.<b>qq</b>.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET  
to:  
    https://qyapi.weixin.<b>ultragis</b>.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET  
and access_token can be shared by multiple process or multiple server.  
  
Set wechat callback address like this:  
    https://svc.weixin.ultragis.com/?<b>call</b>=https%3A//api.weixin.ultragis.com&<b>call</b>=www.ultragis.com  
wechat callback message will dispatch to https://api.weixin.ultragis.com and http://www.ultragis.com  
  
If multiple call address has been set, first none empty result will be return to wechat server.



## 微信代理  
用于全局缓存微信 access_token 和转发微信回调消息的服务器程序。  
自动缓存微信的 access_token 并将微信回调消息转发至多个后台服务器。  
<br/>  
**微信公众号**  
只需简单地替换 access_token 请求网址:  <br/>
    https://api.weixin.<b>qq</b>.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET  <br/>
为:  <br/>
    https://api.weixin.<b>ultragis</b>.com/cgi-bin/token?grant_type=client_credential&appid=APPID&secret=SECRET  <br/>
即可多个进程或多台服务器共享 access_token。<br/>  
  <br/>
**微信企业号**  <br/>
只需简单地替换 access_token 请求网址：  <br/>
    https://qyapi.weixin.<b>qq</b>.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET   <br/>
为:   <br/>
    https://qyapi.weixin.<b>ultragis</b>.com/cgi-bin/gettoken?corpid=CORPID&corpsecret=SECRET   <br/>
即可多个进程或多台服务器共享 access_token。  <br/>
  <br/>
**微信回调消息**  <br/>
设置微信回调消息网址为：  <br/>
    https://svc.weixin.ultragis.com/?<b>call</b>=https%3A//api.weixin.ultragis.com&<b>call</b>=www.ultragis.com   <br/>
微信回调消息将自动转发给 https://api.weixin.ultragis.com 和 http://www.ultragis.com  <br/>
  <br/>
如果设置了多个 call 参数，系统会将第一个非空的请求结果返回给微信服务器。  <br/>
