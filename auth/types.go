package auth

import (
	"context"
	"errors"
)

const (
	// ProviderWechatMini 表示微信小程序登录来源。
	ProviderWechatMini = "wechat_mini"
	// ProviderAlipayMini 表示支付宝小程序登录来源。
	ProviderAlipayMini = "alipay_mini"
)

// Identity 用来描述第三方登录场景下的外部身份信息。
type Identity struct {
	Provider    string
	UnionID     string
	OpenID      string
	PhoneNumber string
	ClientIP    string
}

// Validate 用来校验身份信息至少包含可识别的第三方主键。
func (i Identity) Validate() error {
	if i.UnionID == "" && i.OpenID == "" {
		return errors.New("缺少 UnionID 或 OpenID")
	}
	return nil
}

// UserHandler 约定业务方在登录流程中需要实现的用户查找、创建与发令牌逻辑。
type UserHandler interface {
	FindUser(ctx context.Context, identity Identity) (user any, exists bool, err error)
	CreateUser(ctx context.Context, identity Identity) (user any, err error)
	GenerateToken(ctx context.Context, user any, identity Identity, sessionValue string) (data any, err error)
}
