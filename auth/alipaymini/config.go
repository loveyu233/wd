package alipaymini

// Config 用来描述支付宝小程序登录服务所需的配置。
type Config struct {
	AppID                string
	AppPrivateKey        string
	AESKey               string
	AppPublicKeyFilePath string
	AliPublicKeyFilePath string
	AliRootKeyFilePath   string
	SaveHandlerLog       bool
}
