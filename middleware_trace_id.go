package wd

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MiddlewareTraceID 用来确保请求拥有统一的 Trace ID。
func MiddlewareTraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := uuid.NewString()
		c.Header(HeaderTraceID, traceID)
		c.Set(HeaderTraceID, traceID)
		c.Next()
	}
}

func GetTraceID(c *gin.Context) string {
	return c.GetString(HeaderTraceID)
}
