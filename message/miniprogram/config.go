package miniprogram

import (
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	miniapp "github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
)

// Config 用来描述小程序订阅消息服务的初始化配置。
type Config struct {
	AppID             string
	Secret            string
	MessageToken      string
	MessageAESKey     string
	VirtualPayAppKey  string
	VirtualPayOfferID string
	Log               miniapp.Log
	Cache             kernel.CacheInterface
	HTTPDebug         bool
	Debug             bool
}
