package alipaymini

import (
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
)

// RegisterRoutes 用来注册支付宝小程序登录相关路由。
func (s *Service) RegisterRoutes(r *gin.RouterGroup) {
	r.Use(wd.GinLogSetModuleName("支付宝小程序登录"))
	r.POST("/login", wd.GinLogSetOptionName("支付宝小程序登录", s.saveHandlerLog), s.login)
}
