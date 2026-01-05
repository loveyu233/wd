package msg

import (
	"context"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/basicService/subscribeMessage/request"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	"github.com/loveyu233/wd/login"
)

type MiniProgramStateType string

const (
	// MiniProgramStateDeveloper 体验版
	MiniProgramStateDeveloper MiniProgramStateType = "developer"
	// MiniProgramStateTrial 开发板
	MiniProgramStateTrial MiniProgramStateType = "trial"
	// MiniProgramStateFormal 正式版
	MiniProgramStateFormal MiniProgramStateType = "formal"
)

type WxMiniMsgContent struct {
	ToUserOpenID     string
	TemplateID       string
	Page             string
	MiniProgramState MiniProgramStateType
	Data             map[string]map[string]any
}

func WxMiniSubscribeMessageSend(wxMiniApp *login.WXMini, ctx context.Context, content WxMiniMsgContent) (*response.ResponseMiniProgram, error) {
	var data = make(power.HashMap)
	for k, v := range content.Data {
		for k1, v1 := range v {
			data[k] = power.HashMap{
				k1: v1,
			}
		}
	}

	resp, err := wxMiniApp.MiniProgramApp.SubscribeMessage.Send(ctx, &request.RequestSubscribeMessageSend{
		ToUser:           content.ToUserOpenID,
		TemplateID:       content.TemplateID,
		Page:             content.Page,
		MiniProgramState: string(content.MiniProgramState),
		Lang:             "zh_CN",
		Data:             &data,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}
