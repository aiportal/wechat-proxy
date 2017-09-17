# wechat-proxy (微信代理服务)

全局缓存微信 access_token, jsapi ticket等。  
多路转发微信回调消息。  
简化微信 oauth2 认证流程。  
简化微信支付流程。  
简化微信 JSSDK 签名流程。  

## 示例

> 注册app:
  
  https://wx.aiportal.net/register?key=test&appid=wx06766a90ab72960e&secret=05bd8b6064a9941b72ee44d5b3bfdb6a

> access_token:   
  
  https://wx.aiportal.net/app/test/api  
  https://wx.aiportal.net/app/test/api/new
  
> OAuth2:  
  
  首先关注测试号：[微信测试号](http://mmbiz.qpic.cn/mmbiz_jpg/lgEc2N7A7WB5fepEujANMWCLDLGCZjKX2EqjWXObAMN85Jdo7L4h8MuMpecvQWicViawn7nW3YlcRmvzhNjGLscA/0)  
  然后用微信打开链接：[https://wx.aiportal.net/app/test/auth/info?call=/echo](https://wx.aiportal.net/qrcode?path=https%3A%2F%2Fwx.aiportal.net%2Fapp%2Ftest%2Fauth%3Fcall%3D%2Fecho)

> 微信扫码支付:
  
    <img src="https://wx.aiportal.net/app/aiportal/pay/qrcode?fee=1&name=支付测试&call=/echo"><img>

> 微信公众号支付：
  
    var openid = 'o62SMjlZ378PMI6j5b5x8HAoX9YA';
    var url = 'http://' + location.host + '/app/aiportal/pay/js?fee=1&call=/echo&openid=' + openid;
    function test_pay() {
        Vue.http.get(url).then(function (res) {
            var config = res.body;
            alert(config.package);
            pop_pay(config);
        }, function (res) {
            alert('error');
        });
    }

    function pop_pay(config) {
        // 弹出微信支付界面
        WeixinJSBridge.invoke('getBrandWCPayRequest', config, function (res) {
            alert(res.err_msg);
        });
    }
    
  参考页面: <https://wx.aiportal.net/example/jspay.html>

> 微信JSSDK配置：

    <script src="/app/test/js/config?debug=true"></script>

  参考页面：<https://wx.aiportal.net/example/jsapi.html>

## 使用说明：

### 1、公众号注册：
    /register?key=...&appid=...&secret=...
    &token=&aes=
    &mch_id=&mch_key=&server_ip=
	&expires=&call=/msg&call=/api&call=...

参数说明：  
 > key: 自定义的app名称，支持中文，也可以是随机生成的字符串。(必填)   
 > appid: 微信公众号的 appid。(必填)  
 > secret: 微信公众号的 secret。(必填)  
 > token, aes: 用于微信回调消息加解密的秘钥。(/msg接口)  
 如果设置了此项参数，后台应用可以直接以json明文格式接收和回复微信回调消息。(/msg/json接口)   
 > mch_id, mch_key, server_ip: 用于微信支付的账号、秘钥和服务器IP。(/pay接口)
 如果设置了此项参数, 可以使用简单的 url 请求实现微信支付功能。  
 > expires: 过期时间，单位秒。如果设置此项参数，注册信息会在到期后自动删除。
 > call: 可用API，可以重复多次。如果设置此项参数，该app注册信息仅可用于已列出的api接口。
 
### 2、access_token 全局缓存:
access_token 全局缓存自动获取最新的 access_token 值缓存在代理服务器内存中。  
access_token 全局缓存支持多进程、多服务器共享 access_token，还可以无限次获取，简化后台服务的开发难度。

> 调用/register接口完成注册后，使用已注册的 test 名称调用 /api 接口:

    /app/test/api
    /app/test/qyapi

> 强制刷新 access_token:  

    /app/test/api/new
    /app/test/qyapi/new

### 3、微信回调消息的多路转发：  

微信回调消息的多路转发可以将微信公众号的回调消息转发给多个后台服务，按照call参数的设置顺序返回第一个非空的处理结果。  
如果在/register接口中设置了token和aes参数，/msg/json 接口支持微信消息的自动加解密服务，后台call网址可直接使用 json 明文协议实现交互。    

    /app/test/msg?call=...&call=...  
    /app/test/msg/json?call=...&call=...

### 4、微信登录:

> snsapi_base 方式登录验证：  
    
    /app/test/auth?call=...&state=&lang=

> snsapi_info 方式登录验证：

    /app/test/auth/info?call=...&state=&lang=

> 验证成功时，call网址将收到 json 数据包(POST)，包含用户的 openid, unionid, 以及用户的其他信息。  
> state和lang是可选参数，具体含义请参考微信官方文档。  

### 5、微信支付：

>微信支付二维码：直接返回二维码图片，用户使用微信扫码后即可付款。
    
    /app/test/pay/qrcode?fee=...&name=&call=&...
    
>公众号网页支付：公众号网页内调起支付窗口完成支付。(参考实现：/example/jspay.html)
    
    /app/test/pay/js?openid=...&fee=...&name=&call=&...

>统一下单：服务端可调用统一下单接口获得支付订单，省去签名计算等步骤。

    /app/test/pay?fee=...&name=&call=&...

参数说明：
> fee: 订单金额，单位分。(必填)  
> openid: 用户在该公众号下的 openid。(网页支付必填) 允许使用客户端 cookie 传递此参数。  
> name: 订单名称。
> call: 回调通知网址。订单支付成功后将支付结果发送至此网址。(JSON)  
> 其他参数(高级用法)：支持[微信统一下单接口](https://www.google.com.hk/url?sa=t&rct=j&q=&esrc=s&source=web&cd=1&ved=0ahUKEwiToMqf1aPWAhWLxrwKHZEMBXEQFggnMAA&url=https%3A%2F%2Fpay.weixin.qq.com%2Fwiki%2Fdoc%2Fapi%2Fjsapi.php%3Fchapter%3D9_1&usg=AFQjCNEaVYHJTMZBzBO8zk_BbWFVCKfXwQ)
所列举的其他订单参数。具体请参考微信官方文档。(sign, sign_type 由程序自动生成，不可覆盖)

### 6、JSSDK：

> jsapi_ticket 全局缓存：

    /app/test/jsapi

> JSSDK 权限验证配置：直接返回 wx.config({...}); 默认获取全部API权限。  
> 可选参数：  
> debug: true或false。  
> apilist: 逗号分隔的 JSSDK API 列表。

    <script src="/app/test/js/config?debug=true"></script>

> 微信卡券签名：
   
    /app/test/js/card
   
