package wechatmini

import (
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
)

// Config 用来描述微信小程序登录与二维码相关能力所需的配置。
type Config struct {
	AppID             string
	Secret            string
	MessageToken      string
	MessageAESKey     string
	VirtualPayAppKey  string
	VirtualPayOfferID string
	Log               miniProgram.Log
	Cache             kernel.CacheInterface
	HTTPDebug         bool
	Debug             bool
	SaveHandlerLog    bool
}
