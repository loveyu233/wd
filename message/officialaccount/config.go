package officialaccount

import "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"

// Config 用来描述微信公众号消息服务所需的配置。
type Config struct {
	AppID          string
	AppSecret      string
	MessageToken   string
	MessageAESKey  string
	ResponseType   string
	Cache          kernel.CacheInterface
	HTTPDebug      bool
	SaveHandlerLog bool
}
