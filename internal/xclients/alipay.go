package xclients

import (
	"errors"

	"github.com/go-pay/gopay/alipay/v3"
	wd "github.com/loveyu233/wd"
)

// AlipayConfig 用来统一描述支付宝客户端初始化参数。
type AlipayConfig struct {
	AppID                string
	AppPrivateKey        string
	AESKey               string
	AppPublicKeyFilePath string
	AliPublicKeyFilePath string
	AliRootKeyFilePath   string
}

// AlipayBundle 用来聚合支付宝客户端和证书内容。
type AlipayBundle struct {
	Client       *alipay.ClientV3
	AppPublicKey []byte
	AliPublicKey []byte
	AliRootKey   []byte
}

// NewAlipayBundle 用来初始化支付宝客户端并加载证书内容。
func NewAlipayBundle(config AlipayConfig) (*AlipayBundle, error) {
	if config.AppID == "" || config.AppPrivateKey == "" {
		return nil, errors.New("支付宝 AppID 或应用私钥不能为空")
	}
	appPublicKey, err := wd.ReadFileContent(config.AppPublicKeyFilePath)
	if err != nil {
		return nil, err
	}
	aliPublicKey, err := wd.ReadFileContent(config.AliPublicKeyFilePath)
	if err != nil {
		return nil, err
	}
	aliRootKey, err := wd.ReadFileContent(config.AliRootKeyFilePath)
	if err != nil {
		return nil, err
	}
	clientV3, err := alipay.NewClientV3(config.AppID, config.AppPrivateKey, true)
	if err != nil {
		return nil, err
	}
	if config.AESKey != "" {
		clientV3.SetAESKey(config.AESKey)
	}
	if err = clientV3.SetCert(appPublicKey, aliPublicKey, aliRootKey); err != nil {
		return nil, err
	}
	return &AlipayBundle{
		Client:       clientV3,
		AppPublicKey: appPublicKey,
		AliPublicKey: aliPublicKey,
		AliRootKey:   aliRootKey,
	}, nil
}
