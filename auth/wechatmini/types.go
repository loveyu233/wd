package wechatmini

import "context"

// LoginRequest 表示微信小程序登录接口的请求体。
type LoginRequest struct {
	Code          string `json:"code" binding:"required"`
	EncryptedData string `json:"encrypted_data"`
	IvStr         string `json:"iv_str"`
}

// Phone 用来承接微信解密出的手机号信息。
type Phone struct {
	PhoneNumber string `json:"phoneNumber"`
}

// MiniCode 表示生成小程序码时的参数。
type MiniCode struct {
	ctx        context.Context
	pagePath   string
	width      int64
	r, g, b    int64
	envVersion string
	autoColor  bool
	isHyaline  bool
}

// MiniUnlimitedCode 表示不限量小程序码的生成参数。
type MiniUnlimitedCode struct {
	ctx        context.Context
	pagePath   string
	scene      string
	width      int64
	r, g, b    int64
	envVersion string
	autoColor  bool
	isHyaline  bool
	checkPage  bool
}
