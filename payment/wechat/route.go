package wechat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
)

// RegisterRoutes 用来注册微信支付相关路由。
func (s *Service) RegisterRoutes(r *gin.RouterGroup) {
	r.Use(wd.GinLogSetModuleName("微信支付"))
	if s.handler != nil {
		r.POST("/notify/payment", wd.GinLogSetOptionName("支付异步回调", s.saveHandlerLog), s.wxPayCallback)
		r.POST("/notify/refund", wd.GinLogSetOptionName("退款异步回调", s.saveHandlerLog), s.wxRefundCallback)
	}
	r.POST("/pay", wd.GinLogSetOptionName("支付请求", s.saveHandlerLog), s.pay)
	r.POST("/refund", wd.GinLogSetOptionName("退款请求", s.saveHandlerLog), s.refund)
}

func writeCallbackFailure(c *gin.Context) {
	c.String(http.StatusInternalServerError, "fail")
}
