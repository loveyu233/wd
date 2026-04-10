package xclients

import (
	"errors"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
)

// MiniProgramConfig 用来统一描述微信小程序客户端初始化参数。
type MiniProgramConfig struct {
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
}

// NewMiniProgramClient 用来按统一配置初始化微信小程序客户端。
func NewMiniProgramClient(config MiniProgramConfig) (*miniProgram.MiniProgram, error) {
	if config.AppID == "" || config.Secret == "" {
		return nil, errors.New("微信小程序 AppID 或 Secret 不能为空")
	}
	return miniProgram.NewMiniProgram(&miniProgram.UserConfig{
		AppID:        config.AppID,
		Secret:       config.Secret,
		ResponseType: response.TYPE_MAP,
		Token:        config.MessageToken,
		AESKey:       config.MessageAESKey,
		AppKey:       config.VirtualPayAppKey,
		OfferID:      config.VirtualPayOfferID,
		Log:          config.Log,
		Cache:        config.Cache,
		HttpDebug:    config.HTTPDebug,
		Debug:        config.Debug,
	})
}
