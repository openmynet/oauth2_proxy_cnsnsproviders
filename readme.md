# oauth2_proxy 国内的providers
---
> 当前已粗略的实现wechat

## oauth2_proxy 配置
将 providers 复制到 oauth2_proxy 项目中并覆盖 
注： 当前支持 oauth2_proxy `v4.1.0` 版本

oauth2_proxy项目:  https://github.com/pusher/oauth2_proxy

## 部署配置参考
> 具体部署参考 www 没目录下的文件

1. www/oauth.html 微信中打开
2. nginx.conf 
3. confg.cfg   oauth2_proxy的配置文件


## TODO
1. [x] wechat
2. [ ] qq
3. [ ] weibo
4. [ ] 支付宝