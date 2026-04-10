package alipay

import (
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
)

// RegisterRoutes 用来注册支付宝支付相关路由。
func (s *Service) RegisterRoutes(r *gin.RouterGroup) {
	r.Use(wd.GinLogSetModuleName("支付宝支付"))
	r.POST("/notify", wd.GinLogSetOptionName("支付宝支付异步回调", s.saveHandlerLog), s.notify)
	r.POST("/pay", wd.GinLogSetOptionName("支付宝支付请求", s.saveHandlerLog), s.pay)
	r.POST("/refund", wd.GinLogSetOptionName("支付宝退款请求", s.saveHandlerLog), s.refund)
}
