package wd

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

type HTTPServer struct {
	server *http.Server
}

var (
	globalApiPrefix string
)

// InitHTTPServerAndStart 用来根据路由配置启动 HTTP 服务并注册钩子。如果初始化了GinJWTMiddleware则默认会添加上
func InitHTTPServerAndStart(listenAddr string, opts ...GinRouterConfigOptionFunc) *HTTPServer {
	var config RouterConfig
	for _, opt := range opts {
		opt(&config)
	}
	if InsGinJWTMiddleware != nil && !config.skipGinJWTMiddleware {
		config.authMiddleware = append(config.authMiddleware, InsGinJWTMiddleware.MiddlewareFunc())
	}
	if config.model == "" {
		config.model = GinModelDebug
	}
	if config.prefix == "" {
		config.prefix = "/api"
	}
	globalApiPrefix = config.prefix
	engine := initPrivateRouter(config)
	server := &HTTPServer{server: &http.Server{
		Addr:    listenAddr,
		Handler: engine,
	}}
	if config.engineFunc != nil {
		config.engineFunc(engine)
	}
	if config.readTimeout > 0 {
		server.server.ReadTimeout = config.readTimeout
	}
	if config.writeTimeout > 0 {
		server.server.WriteTimeout = config.writeTimeout
	}
	if config.idleTimeout > 0 {
		server.server.IdleTimeout = config.idleTimeout
	}
	if config.maxHeaderBytes > 0 {
		server.server.MaxHeaderBytes = config.maxHeaderBytes
	}
	go server.startHTTPServer()
	server.setupGracefulShutdown()
	return server
}

// startHTTPServer 用来启动底层 http.Server。
func (h *HTTPServer) startHTTPServer() {
	if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(err)
	}
}

// setupGracefulShutdown 用来注册系统信号以优雅关闭服务。
func (h *HTTPServer) setupGracefulShutdown() {
	InsGlobalHook.AppendFun(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		if err := h.server.Shutdown(ctx); err != nil {
			log.Printf("http close err: %s\n", err)
		} else {
			log.Printf("http close success\n")
		}
	})

	InsGlobalHook.Close()
}
