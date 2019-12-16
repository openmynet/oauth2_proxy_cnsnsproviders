package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pusher/oauth2_proxy/pkg/apis/sessions"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type WechatProvider struct {
	*ProviderData
}

/**
1.大多数情况下需要实现 Configure、GetLoginURL、Redeem这三个方法
 */

// NewWechatProvider initiates a new WechatProvider
func NewWechatProvider(p *ProviderData) *WechatProvider {
	p.ProviderName = "Wechat"
	// 配置获取用户信息的地址
	if p.ProfileURL == nil || p.ProfileURL.String() == "" {
		p.ProfileURL = &url.URL{
			Scheme: "https",
			Host:   "api.weixin.qq.com",
			Path:   "/sns/userinfo",
		}
		// access_token=?&openid=?
	}
	// 登陆地址
	if p.LoginURL == nil || p.LoginURL.String() == "" {
		p.LoginURL = &url.URL{
			Scheme: "https",
			Host:   "open.weixin.qq.com",
			Path:   "/connect/oauth2/authorize"}
	}
	// token取回地址
	if p.RedeemURL == nil || p.RedeemURL.String() == "" {
		p.RedeemURL = &url.URL{
			Scheme: "https",
			Host:   "api.weixin.qq.com",
			Path:   "/sns/oauth2/access_token",
		}
	}
	// 受保护的资源??微信下是否有必要
	if p.ProtectedResource == nil || p.ProtectedResource.String() == "" {
		p.ProtectedResource = &url.URL{
			Scheme: "https",
			Host:   "api.weixin.qq.com",
		}
	}
	if p.Scope == "" {
		p.Scope = "snsapi_userinfo"
	}
	return &WechatProvider{ProviderData: p}
}

func (p *WechatProvider) GetLoginURL(redirectURI, state string) string {
	var a url.URL
	a = *p.LoginURL
	params, _ := url.ParseQuery(a.RawQuery)
	params.Set("redirect_uri", redirectURI)
	params.Add("scope", p.Scope)
	params.Set("appid", p.ClientID)
	params.Set("response_type", "code")
	params.Add("state", state)
	a.RawQuery = params.Encode()
	return a.String()
}

func (p *WechatProvider) Redeem(redirectURL, code string) (s *sessions.SessionState, err error) {
	// 用于获取account_token
	if code == "" {
		err = errors.New("missing code")
		return
	}
	params := url.Values{}
	params.Add("appid", p.ClientID)
	params.Add("secret", p.ClientSecret)
	params.Add("code", code)
	params.Add("grant_type", "authorization_code")
	redeemURL := p.RedeemURL.String() + "?" + params.Encode()
	var req *http.Request
	req, err = http.NewRequest("GET", redeemURL, nil)
	if err != nil {
		return
	}
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("got %d from %q %s", resp.StatusCode, p.RedeemURL.String(), body)
		return
	}

	var jsonResponse struct {
		AccessToken  string `json:"access_token"`  // 网页授权接口调用凭证
		RefreshToken string `json:"refresh_token"` // 刷新 access_token 的凭证
		ExpiresIn    int64  `json:"expires_in"`    // access_token 接口调用凭证超时时间, 单位: 秒
		IDToken      string `json:"id_token"`      // jwt格式，微信认证不存在此功能的字段

		OpenId  string `json:"openid,omitempty"`  // openID
		UnionId string `json:"unionid,omitempty"` // 用户唯一编码，相同主体不同服务号
		Scope   string `json:"scope,omitempty"`   // 用户授权的作用域, 使用逗号(,)分隔
	}

	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return
	}
	email := fmt.Sprintf("%s@%s.%s", jsonResponse.OpenId, p.ClientID, p.ProviderName)
	s = &sessions.SessionState{
		AccessToken:  jsonResponse.AccessToken,
		IDToken:      jsonResponse.IDToken,
		CreatedAt:    time.Now(),
		ExpiresOn:    time.Now().Add(time.Duration(jsonResponse.ExpiresIn - 10) * time.Second), // -10s 的缓冲时间
		RefreshToken: jsonResponse.RefreshToken,
		Email:        email,
	}
	return
}

// GetEmailAddress returns the Account email address
func (p *WechatProvider) GetEmailAddress(s *sessions.SessionState) (string, error) {
	return s.Email, nil
}
func (p *WechatProvider) GetUserName(s *sessions.SessionState) (name string, err error) {
	openid := strings.Split(s.Email, "@")
	name = openid[0]
	params := url.Values{}
	params.Add("access_token", s.AccessToken)
	params.Add("openid", openid[0])
	params.Add("lang", "zh_CN")
	profileURL := p.ProfileURL.String() + "?" + params.Encode()

	var req *http.Request
	req, err = http.NewRequest("GET", profileURL, nil)
	if err != nil {
		return
	}
	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return name, err
	}
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return
	}

	if resp.StatusCode != 200 {
		err = fmt.Errorf("got %d from %q %s", resp.StatusCode, p.RedeemURL.String(), body)
		return
	}
	var jsonResponse struct {
		OpenId   string `json:"openid"`   // 用户的唯一标识
		Nickname string `json:"nickname"` // 用户昵称
		Sex      int    `json:"sex"`      // 用户的性别, 值为1时是男性, 值为2时是女性, 值为0时是未知
		City     string `json:"city"`     // 普通用户个人资料填写的城市
		Province string `json:"province"` // 用户个人资料填写的省份
		Country  string `json:"country"`  // 国家, 如中国为CN

		// 用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），
		// 用户没有头像时该项为空。若用户更换头像，原有头像URL将失效。
		HeadImageURL string `json:"headimgurl,omitempty"`

		Privilege []string `json:"privilege,omitempty"` // 用户特权信息，json 数组，如微信沃卡用户为（chinaunicom）
		UnionId   string   `json:"unionid,omitempty"`   // 只有在用户将公众号绑定到微信开放平台帐号后，才会出现该字段。
	}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return
	}
	name = fmt.Sprintf(`{"name":"%s","avatar":"%s"}`,jsonResponse.Nickname, jsonResponse.HeadImageURL)
	return
}
