package wd

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Cors 用来为 gin 路由添加通用的跨域响应头。
// 传入允许的 Origin 白名单；不传则允许所有 Origin（仅适用于开发环境）。
func Cors(allowedOrigins ...string) gin.HandlerFunc {
	const (
		allowMethods  = "POST, GET, OPTIONS, PUT, DELETE"
		allowHeaders  = "Content-Type, Authorization, X-Requested-With"
		exposeHeaders = "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers"
		maxAge        = "86400"
	)

	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader(HeaderOrigin)

		if len(originSet) > 0 {
			// 白名单模式
			if _, ok := originSet[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
			} else {
				// Origin 不在白名单中，不设置 CORS 头
				if strings.EqualFold(c.Request.Method, http.MethodOptions) {
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				c.Next()
				return
			}
		} else {
			// 未配置白名单，允许所有（开发模式）
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Vary", "Origin")
				c.Header("Access-Control-Allow-Credentials", "true")
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
				c.Header("Access-Control-Allow-Credentials", "false")
			}
		}

		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)
		c.Header("Access-Control-Expose-Headers", exposeHeaders)
		c.Header("Access-Control-Max-Age", maxAge)

		if strings.EqualFold(c.Request.Method, http.MethodOptions) {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
