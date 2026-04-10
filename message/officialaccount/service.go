package officialaccount

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/contract"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/messages"
	models2 "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount"
	broadcastrequest "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/broadcasting/request"
	servermodels "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/server/handlers/models"
	templaterequest "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/templateMessage/request"
	templateresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/templateMessage/response"
	userresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/officialAccount/user/response"
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
	"github.com/loveyu233/wd/internal/xhelper"
)

// Handler 约定业务方需要提供的公众号消息处理逻辑。
type Handler interface {
	OnSubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
	OnUnsubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
	BuildPushRequest(c *gin.Context) (toUsers []string, message string, err error)
}

// Service 聚合微信公众号回调与主动推送能力。
type Service struct {
	client         *officialAccount.OfficialAccount
	handler        Handler
	saveHandlerLog bool
}

// New 用来根据配置初始化公众号消息服务。
func New(config Config, handler Handler) (*Service, error) {
	if config.AppID == "" || config.AppSecret == "" {
		return nil, errors.New("微信公众号 AppID 或 AppSecret 不能为空")
	}
	app, err := officialAccount.NewOfficialAccount(&officialAccount.UserConfig{
		AppID:        config.AppID,
		Secret:       config.AppSecret,
		Token:        config.MessageToken,
		AESKey:       config.MessageAESKey,
		ResponseType: config.ResponseType,
		Cache:        config.Cache,
		HttpDebug:    config.HTTPDebug,
	})
	if err != nil {
		return nil, err
	}
	return NewWithClient(app, handler, config.SaveHandlerLog)
}

// NewWithClient 用来复用外部传入的公众号客户端。
func NewWithClient(client *officialAccount.OfficialAccount, handler Handler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("微信公众号客户端不能为空")
	}
	if xhelper.IsNil(handler) {
		return nil, errors.New("微信公众号处理器不能为空")
	}
	return &Service{client: client, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层公众号客户端。
func (s *Service) Client() *officialAccount.OfficialAccount {
	return s.client
}

func (s *Service) callbackVerify(c *gin.Context) {
	rs, err := s.client.Server.VerifyURL(c.Request)
	if err != nil {
		c.String(http.StatusBadRequest, "verify failed")
		return
	}
	defer rs.Body.Close()
	text, err := io.ReadAll(rs.Body)
	if err != nil {
		log.Printf("[微信公众号] 读取回调验证响应失败: %v", err)
		c.String(http.StatusInternalServerError, "verify failed")
		return
	}
	c.String(http.StatusOK, string(text))
}

func (s *Service) callback(c *gin.Context) {
	rs, err := s.client.Server.Notify(c.Request, func(event contract.EventInterface) interface{} {
		ctx := c.Request.Context()
		switch event.GetMsgType() {
		case models2.CALLBACK_MSG_TYPE_TEXT:
			msg := servermodels.MessageText{}
			if err := event.ReadMessage(&msg); err != nil {
				log.Printf("[微信公众号] 读取文本消息失败: %v", err)
				return "error"
			}
		case models2.CALLBACK_MSG_TYPE_EVENT:
			user, err := s.client.User.Get(ctx, event.GetFromUserName(), "zh_CN")
			if err != nil {
				log.Printf("[微信公众号] 获取用户信息失败: %v", err)
				return "error"
			}
			switch event.GetEvent() {
			case "subscribe":
				if user.OpenID != "" {
					if err := s.handler.OnSubscribe(ctx, user, event); err != nil {
						log.Printf("[微信公众号] 关注回调处理失败: %v", err)
						return "error"
					}
				}
				return messages.NewText("感谢您的关注！")
			case "unsubscribe":
				if err := s.handler.OnUnsubscribe(ctx, user, event); err != nil {
					log.Printf("[微信公众号] 取消关注回调处理失败: %v", err)
					return "error"
				}
			}
		}
		return ""
	})
	if err != nil {
		log.Printf("[微信公众号] 回调通知处理失败: %v", err)
		c.String(http.StatusInternalServerError, "fail")
		return
	}
	defer rs.Body.Close()
	text, err := io.ReadAll(rs.Body)
	if err != nil {
		log.Printf("[微信公众号] 读取回调响应失败: %v", err)
		c.String(http.StatusInternalServerError, "fail")
		return
	}
	c.String(http.StatusOK, string(text))
}

func (s *Service) push(c *gin.Context) {
	users, message, err := s.handler.BuildPushRequest(c)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrBadRequest("推送参数错误", err))
		return
	}
	if _, err = s.Push(c.Request.Context(), users, message); err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechat("消息推送失败，请稍后重试", err))
		return
	}
	wd.ResponseSuccessMsg(c, "发送成功")
}

// Push 用来群发文本消息。
func (s *Service) Push(ctx context.Context, toUser []string, message string) (any, error) {
	if len(toUser) == 1 {
		toUser = append(toUser, "")
	}
	return s.client.Broadcasting.SendText(ctx, message, &broadcastrequest.Reception{
		ToUser: toUser,
		Filter: &broadcastrequest.Filter{IsToAll: false, TagID: 0},
	}, &power.HashMap{})
}

// PushTemplateMessage 用来发送模板消息。
func (s *Service) PushTemplateMessage(ctx context.Context, toUser, templateID string, data any) (*templateresponse.ResponseTemplateSend, error) {
	templateData, err := structToMapWithJSONTag(data)
	if err != nil {
		return nil, err
	}
	payload := make(power.HashMap, len(templateData))
	for key, value := range templateData {
		payload[key] = &power.HashMap{"value": value, "color": "#173177"}
	}
	return s.client.TemplateMessage.Send(ctx, &templaterequest.RequestTemlateMessage{
		ToUser:     toUser,
		TemplateID: templateID,
		Data:       &payload,
	})
}

func structToMapWithJSONTag(obj any) (map[string]any, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
