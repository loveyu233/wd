package miniprogram

import (
	"context"
	"errors"

	subscriberequest "github.com/ArtisanCloud/PowerWeChat/v3/src/basicService/subscribeMessage/request"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	miniapp "github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
	"github.com/loveyu233/wd/internal/xclients"
)

// Service 聚合小程序订阅消息发送能力。
type Service struct {
	client *miniapp.MiniProgram
}

// New 用来根据配置初始化小程序消息服务。
func New(config Config) (*Service, error) {
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
	return NewWithClient(app)
}

// NewWithClient 用来复用外部传入的小程序客户端。
func NewWithClient(client *miniapp.MiniProgram) (*Service, error) {
	if client == nil {
		return nil, errors.New("小程序客户端不能为空")
	}
	return &Service{client: client}, nil
}

// Client 用来返回底层小程序客户端。
func (s *Service) Client() *miniapp.MiniProgram {
	return s.client
}

// SubscribeMessageSend 用来发送小程序订阅消息。
func (s *Service) SubscribeMessageSend(ctx context.Context, content SubscribeContent) (*response.ResponseMiniProgram, error) {
	data := make(power.HashMap, len(content.Data))
	for key, value := range content.Data {
		item := make(power.HashMap, len(value))
		for fieldKey, fieldValue := range value {
			item[fieldKey] = fieldValue
		}
		data[key] = item
	}
	return s.client.SubscribeMessage.Send(ctx, &subscriberequest.RequestSubscribeMessageSend{
		ToUser:           content.ToUserOpenID,
		TemplateID:       content.TemplateID,
		Page:             content.Page,
		MiniProgramState: string(content.State),
		Lang:             "zh_CN",
		Data:             &data,
	})
}
