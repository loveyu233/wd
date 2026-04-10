package officialaccount

import (
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
)

// RegisterRoutes 用来注册微信公众号消息相关路由。
func (s *Service) RegisterRoutes(r *gin.RouterGroup) {
	r.Use(wd.GinLogSetModuleName("微信公众号消息"))
	r.GET("/callback", wd.GinLogSetOptionName("公众号回调验证", s.saveHandlerLog), s.callbackVerify)
	r.POST("/callback", wd.GinLogSetOptionName("公众号收到消息", s.saveHandlerLog), s.callback)
	r.POST("/push", wd.GinLogSetOptionName("公众号推送消息", s.saveHandlerLog), s.push)
}
