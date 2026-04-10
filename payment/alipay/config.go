package alipay

// Config 用来描述支付宝支付服务所需的配置。
type Config struct {
	AppID                string
	AppPrivateKey        string
	AESKey               string
	AppPublicKeyFilePath string
	AliPublicKeyFilePath string
	AliRootKeyFilePath   string
	NotifyURL            string
	SaveHandlerLog       bool
}
