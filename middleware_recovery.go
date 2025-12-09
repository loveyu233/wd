package wd

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// MiddlewareRecovery 用来捕获请求中的 panic 并返回统一错误响应。
func MiddlewareRecovery(log ...CustomLog) gin.HandlerFunc {
	var loclLog CustomLog

	if len(log) > 0 {
		loclLog = log[0]
	} else {
		loclLog = new(CustomDefaultLogger)
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				loclLog.Errorf("panic:%s;stack:%s", err, string(debug.Stack()))
				ResponseError(c, ErrServerBusy.WithMessage("panic:%v", err))
				c.AbortWithStatus(http.StatusOK)
			}
		}()
		c.Next()
	}
}
