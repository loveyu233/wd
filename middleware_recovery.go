package wd

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// MiddlewareRecovery 用来捕获请求中的 panic 并返回统一错误响应。
func MiddlewareRecovery(log ...func(recoverErr any, debugStack []byte)) gin.HandlerFunc {
	var isFunc bool
	if len(log) > 0 && log[0] != nil {
		isFunc = true
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if isFunc {
					log[0](err, debug.Stack())
				}
				ResponseError(c, MsgErrServerBusy("服务异常，请稍后重试"))
				c.AbortWithStatus(http.StatusOK)
			}
		}()
		c.Next()
	}
}
