package auth

import "github.com/gin-gonic/gin"

// RouteRegistrar 表示可向 gin 路由组注册认证相关路由的服务。
type RouteRegistrar interface {
	RegisterRoutes(r *gin.RouterGroup)
}

// Register 用来批量注册认证服务的路由。
func Register(r *gin.RouterGroup, services ...RouteRegistrar) {
	for _, service := range services {
		if service == nil {
			continue
		}
		service.RegisterRoutes(r)
	}
}
