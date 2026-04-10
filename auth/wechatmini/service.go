package wechatmini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
	"github.com/loveyu233/wd/auth"
	"github.com/loveyu233/wd/internal/xclients"
	"github.com/loveyu233/wd/internal/xhelper"
)

// Service 聚合微信小程序认证与码生成能力。
type Service struct {
	client         *miniProgram.MiniProgram
	handler        auth.UserHandler
	saveHandlerLog bool
}

// New 用来根据配置初始化微信小程序服务。
func New(config Config, handler auth.UserHandler) (*Service, error) {
	app, err := xclients.NewMiniProgramClient(xclients.MiniProgramConfig{
		AppID:             config.AppID,
		Secret:            config.Secret,
		MessageToken:      config.MessageToken,
		MessageAESKey:     config.MessageAESKey,
		VirtualPayAppKey:  config.VirtualPayAppKey,
		VirtualPayOfferID: config.VirtualPayOfferID,
		Log:               config.Log,
		Cache:             config.Cache,
		HTTPDebug:         config.HTTPDebug,
		Debug:             config.Debug,
	})
	if err != nil {
		return nil, err
	}
	return NewWithClient(app, handler, config.SaveHandlerLog)
}

// NewWithClient 用来在调用方已持有 MiniProgram 客户端时复用现有实例。
func NewWithClient(client *miniProgram.MiniProgram, handler auth.UserHandler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("微信小程序客户端不能为空")
	}
	if xhelper.IsNil(handler) {
		return nil, errors.New("微信小程序用户处理器不能为空")
	}
	return &Service{client: client, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层 MiniProgram SDK 客户端，便于其他能力复用。
func (s *Service) Client() *miniProgram.MiniProgram {
	return s.client
}

func (s *Service) login(c *gin.Context) {
	var params LoginRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		wd.ResponseError(c, wd.MsgErrInvalidParam(err))
		return
	}

	ctx := c.Request.Context()
	session, err := s.client.Auth.Session(ctx, params.Code)
	if err != nil || session.ErrCode != 0 {
		wd.ResponseError(c, wd.MsgErrRequestWechat("微信登录失败，请重试", err))
		return
	}

	identity := auth.Identity{
		Provider: auth.ProviderWechatMini,
		UnionID:  session.UnionID,
		OpenID:   session.OpenID,
		ClientIP: c.ClientIP(),
	}
	if err := identity.Validate(); err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechat("微信登录信息不完整，请重试", err))
		return
	}

	user, exists, err := s.handler.FindUser(ctx, identity)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrDatabase("用户信息查询失败，请稍后重试", err))
		return
	}
	if !exists {
		if params.EncryptedData == "" {
			wd.ResponseSuccess(c, gin.H{"open_id": session.OpenID})
			return
		}
		phoneInfo, err := s.decryptPhone(params, session.SessionKey)
		if err != nil {
			wd.ResponseError(c, wd.MsgErrRequestWechat("获取手机号失败，请重新授权", err))
			return
		}
		identity.PhoneNumber = phoneInfo.PhoneNumber
		user, err = s.handler.CreateUser(ctx, identity)
		if err != nil {
			wd.ResponseError(c, wd.MsgErrDatabase("注册失败，请稍后重试", err))
			return
		}
	}

	result, err := s.handler.GenerateToken(ctx, user, identity, session.SessionKey)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrServerBusy("登录失败，请稍后重试", err))
		return
	}
	auth.RespondLoginResult(c, result)
}

func (s *Service) decryptPhone(params LoginRequest, sessionKey string) (*Phone, error) {
	if params.EncryptedData == "" || params.IvStr == "" {
		return nil, errors.New("缺少手机号解密参数")
	}
	data, decryptErr := s.client.Encryptor.DecryptData(params.EncryptedData, sessionKey, params.IvStr)
	if decryptErr != nil {
		return nil, fmt.Errorf("微信解密手机号失败")
	}
	var info Phone
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	if info.PhoneNumber == "" {
		return nil, errors.New("手机号为空")
	}
	return &info, nil
}

// CreateQRCode 用来生成小程序二维码。
func (s *Service) CreateQRCode(ctx context.Context, pagePath string, width int64) (*http.Response, error) {
	return s.client.WXACode.CreateQRCode(ctx, pagePath, width)
}

// NewMiniCode 用来创建普通小程序码参数对象。
func NewMiniCode(ctx context.Context) *MiniCode {
	return &MiniCode{ctx: ctx}
}

// SetPagePath 设置扫码进入的小程序页面路径。
func (m *MiniCode) SetPagePath(pagePath string) *MiniCode {
	m.pagePath = pagePath
	return m
}

// SetWidth 设置小程序码宽度。
func (m *MiniCode) SetWidth(width int64) *MiniCode {
	m.width = width
	return m
}

// SetRGB 设置自定义 RGB 颜色。
func (m *MiniCode) SetRGB(r, g, b int64) *MiniCode {
	m.r, m.g, m.b = r, g, b
	return m
}

// SetEnvVersion 设置小程序版本环境。
func (m *MiniCode) SetEnvVersion(version string) *MiniCode {
	m.envVersion = version
	return m
}

// SetAutoColor 设置是否自动配色。
func (m *MiniCode) SetAutoColor(autoColor bool) *MiniCode {
	m.autoColor = autoColor
	return m
}

// SetIsHyaline 设置是否透明底色。
func (m *MiniCode) SetIsHyaline(isHyaline bool) *MiniCode {
	m.isHyaline = isHyaline
	return m
}

// GetCode 用来生成普通小程序码。
func (s *Service) GetCode(code MiniCode) (*http.Response, error) {
	return s.client.WXACode.Get(
		code.ctx,
		code.pagePath,
		code.width,
		code.autoColor,
		&power.HashMap{"r": code.r, "g": code.g, "b": code.b},
		code.isHyaline,
		code.envVersion,
	)
}

// NewMiniUnlimitedCode 用来创建不限量小程序码参数对象。
func NewMiniUnlimitedCode(ctx context.Context) *MiniUnlimitedCode {
	return &MiniUnlimitedCode{ctx: ctx}
}

// SetPagePath 设置页面路径。
func (m *MiniUnlimitedCode) SetPagePath(pagePath string) *MiniUnlimitedCode {
	m.pagePath = pagePath
	return m
}

// SetScene 设置场景参数。
func (m *MiniUnlimitedCode) SetScene(scene string) *MiniUnlimitedCode {
	m.scene = scene
	return m
}

// SetWidth 设置码宽度。
func (m *MiniUnlimitedCode) SetWidth(width int64) *MiniUnlimitedCode {
	m.width = width
	return m
}

// SetRGB 设置自定义颜色。
func (m *MiniUnlimitedCode) SetRGB(r, g, b int64) *MiniUnlimitedCode {
	m.r, m.g, m.b = r, g, b
	return m
}

// SetEnvVersion 设置环境版本。
func (m *MiniUnlimitedCode) SetEnvVersion(version string) *MiniUnlimitedCode {
	m.envVersion = version
	return m
}

// SetAutoColor 设置自动配色。
func (m *MiniUnlimitedCode) SetAutoColor(autoColor bool) *MiniUnlimitedCode {
	m.autoColor = autoColor
	return m
}

// SetIsHyaline 设置是否透明底色。
func (m *MiniUnlimitedCode) SetIsHyaline(isHyaline bool) *MiniUnlimitedCode {
	m.isHyaline = isHyaline
	return m
}

// SetCheckPage 设置是否校验页面存在。
func (m *MiniUnlimitedCode) SetCheckPage(checkPage bool) *MiniUnlimitedCode {
	m.checkPage = checkPage
	return m
}

// GetUnlimitedCode 用来生成不限量小程序码。
func (s *Service) GetUnlimitedCode(code *MiniUnlimitedCode) (*http.Response, error) {
	if code == nil {
		return nil, fmt.Errorf("小程序码参数不能为空")
	}
	return s.client.WXACode.GetUnlimited(
		code.ctx,
		code.scene,
		code.pagePath,
		code.checkPage,
		code.envVersion,
		code.width,
		code.autoColor,
		&power.HashMap{"r": code.r, "g": code.g, "b": code.b},
		code.isHyaline,
	)
}
