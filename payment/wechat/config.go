package wechat

import (
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	wechatpayment "github.com/ArtisanCloud/PowerWeChat/v3/src/payment"
)

// Config 用来描述微信支付服务所需的配置。
type Config struct {
	AppID              string
	MchID              string
	MchApiV3Key        string
	Key                string
	CertPath           string
	KeyPath            string
	SerialNo           string
	WechatPaySerial    string
	CertificateKeyPath string
	RSAPublicKeyPath   string
	SubMchID           string
	SubAppID           string
	NotifyURL          string
	HTTPDebug          bool
	Log                wechatpayment.Log
	HTTP               wechatpayment.Http
	Cache              kernel.CacheInterface
	SaveHandlerLog     bool
}
