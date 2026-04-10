package alipaymini

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay/v3"
	wd "github.com/loveyu233/wd"
	"github.com/loveyu233/wd/auth"
	"github.com/loveyu233/wd/internal/xclients"
	"github.com/loveyu233/wd/internal/xhelper"
)

// Service 聚合支付宝小程序登录能力。
type Service struct {
	client         *alipay.ClientV3
	aesKey         string
	handler        auth.UserHandler
	saveHandlerLog bool
}

// New 用来根据配置初始化支付宝小程序登录服务。
func New(config Config, handler auth.UserHandler) (*Service, error) {
	if xhelper.IsNil(handler) {
		return nil, errors.New("支付宝登录处理器不能为空")
	}
	bundle, err := xclients.NewAlipayBundle(xclients.AlipayConfig{
		AppID:                config.AppID,
		AppPrivateKey:        config.AppPrivateKey,
		AESKey:               config.AESKey,
		AppPublicKeyFilePath: config.AppPublicKeyFilePath,
		AliPublicKeyFilePath: config.AliPublicKeyFilePath,
		AliRootKeyFilePath:   config.AliRootKeyFilePath,
	})
	if err != nil {
		return nil, err
	}
	return &Service{client: bundle.Client, aesKey: config.AESKey, handler: handler, saveHandlerLog: config.SaveHandlerLog}, nil
}

// NewWithClient 用来复用外部传入的支付宝客户端。
func NewWithClient(client *alipay.ClientV3, handler auth.UserHandler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("支付宝客户端不能为空")
	}
	if xhelper.IsNil(handler) {
		return nil, errors.New("支付宝登录处理器不能为空")
	}
	return &Service{client: client, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层支付宝客户端。
func (s *Service) Client() *alipay.ClientV3 {
	return s.client
}

func (s *Service) login(c *gin.Context) {
	var params LoginRequest
	if err := c.ShouldBindJSON(&params); err != nil {
		wd.ResponseError(c, wd.MsgErrInvalidParam(err))
		return
	}

	ctx := c.Request.Context()
	session, err := s.systemOauthToken(ctx, params.Code)
	if err != nil || session.OpenId == "" {
		wd.ResponseError(c, wd.MsgErrRequestAli("支付宝登录失败，请重试", err))
		return
	}

	identity := auth.Identity{
		Provider: auth.ProviderAlipayMini,
		UnionID:  session.UnionId,
		OpenID:   session.OpenId,
		ClientIP: c.ClientIP(),
	}
	if err := identity.Validate(); err != nil {
		wd.ResponseError(c, wd.MsgErrRequestAli("支付宝登录信息不完整，请重试", err))
		return
	}

	user, exists, err := s.handler.FindUser(ctx, identity)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrDatabase("用户信息查询失败，请稍后重试", err))
		return
	}
	if !exists {
		if params.EncryptedData == "" {
			wd.ResponseSuccess(c, gin.H{"open_id": session.OpenId})
			return
		}
		phoneInfo, err := s.mobilePhoneNumberDecryption(params.EncryptedData)
		if err != nil {
			wd.ResponseError(c, wd.MsgErrRequestAli("支付宝授权失败，请重试", err))
			return
		}
		identity.PhoneNumber = phoneInfo.Mobile
		user, err = s.handler.CreateUser(ctx, identity)
		if err != nil {
			wd.ResponseError(c, wd.MsgErrDatabase("注册失败，请稍后重试", err))
			return
		}
	}

	result, err := s.handler.GenerateToken(ctx, user, identity, session.OpenId)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrServerBusy("登录失败，请稍后重试", err))
		return
	}
	auth.RespondLoginResult(c, result)
}

func (s *Service) systemOauthToken(ctx context.Context, code string) (*alipay.SystemOauthTokenRsp, error) {
	body := make(gopay.BodyMap)
	body.Set("grant_type", "authorization_code").Set("code", code)
	return s.client.SystemOauthToken(ctx, body)
}

func (s *Service) mobilePhoneNumberDecryption(response string) (*MobilePhoneNumberDecryptionResp, error) {
	if s.aesKey == "" {
		return nil, errors.New("支付宝 AESKey 不能为空")
	}
	data, err := base64.StdEncoding.DecodeString(s.aesKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(data)
	if err != nil {
		return nil, err
	}
	encryptedBytes, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		return nil, err
	}
	if len(encryptedBytes)%aes.BlockSize != 0 {
		return nil, errors.New("密文长度非法")
	}
	mode := cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize))
	mode.CryptBlocks(encryptedBytes, encryptedBytes)
	decryptedBytes, err := pkcs5Unpad(encryptedBytes)
	if err != nil {
		return nil, err
	}
	var resp MobilePhoneNumberDecryptionResp
	if err = json.Unmarshal(decryptedBytes, &resp); err != nil {
		return nil, err
	}
	if resp.Code != "10000" || resp.Mobile == "" {
		return nil, fmt.Errorf("%s:%s", resp.Msg, resp.SubMsg)
	}
	return &resp, nil
}

func pkcs5Unpad(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, errors.New("解密结果为空")
	}
	unpadding := int(src[len(src)-1])
	if unpadding <= 0 || unpadding > len(src) {
		return nil, errors.New("unpadding error")
	}
	return src[:len(src)-unpadding], nil
}
