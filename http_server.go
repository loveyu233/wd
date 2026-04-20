package wd

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

type HTTPServer struct {
	server    *http.Server
	errCh     chan error
	startOnce sync.Once
}

var (
	globalApiPrefix string
)

// InitHTTPServerAndStart 用来根据路由配置启动 HTTP 服务并注册钩子。如果初始化了GinJWTMiddleware则默认会添加上
func InitHTTPServerAndStart(listenAddr string, opts ...GinRouterConfigOption) *HTTPServer {
	server := NewHTTPServer(listenAddr, opts...)
	server.StartAsync()
	if err := server.Wait(); err != nil {
		log.Printf("http start err: %s\n", err)
	}
	return server
}

// NewHTTPServer 用来创建 HTTP 服务实例并注册优雅关闭逻辑。
func NewHTTPServer(listenAddr string, opts ...GinRouterConfigOption) *HTTPServer {
	var config RouterConfig
	for _, opt := range opts {
		opt(&config)
	}

	if config.model == "" {
		config.model = GinModelDebug
	}
	if config.prefix == "" {
		config.prefix = "/api"
	}
	globalApiPrefix = config.prefix
	engine := initPrivateRouter(config)
	server := &HTTPServer{
		server: &http.Server{
			Addr:    listenAddr,
			Handler: engine,
		},
		errCh: make(chan error, 1),
	}
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
	server.setupGracefulShutdown()
	return server
}

// Start 用来启动底层 http.Server。
func (h *HTTPServer) Start() error {
	if err := h.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// StartAsync 用来异步启动服务，并通过 Err/Wait 暴露启动结果。
func (h *HTTPServer) StartAsync() {
	h.startOnce.Do(func() {
		go func() {
			h.errCh <- h.Start()
			close(h.errCh)
		}()
	})
}

// Err 用来获取服务运行结束时返回的错误。
func (h *HTTPServer) Err() <-chan error {
	return h.errCh
}

// Wait 用来等待启动失败或优雅关闭完成。
func (h *HTTPServer) Wait() error {
	select {
	case err, ok := <-h.errCh:
		if !ok {
			return nil
		}
		if err != nil {
			InsGlobalHook.Trigger()
		}
		return err
	case <-InsGlobalHook.Wait():
		return nil
	}
}

// setupGracefulShutdown 用来注册系统信号以优雅关闭服务。
func (h *HTTPServer) setupGracefulShutdown() {
	InsGlobalHook.AppendFun(func() {
		ctx, cancel := BackgroundTimeout(10 * time.Second)
		defer cancel()
		if err := h.server.Shutdown(ctx); err != nil {
			log.Printf("http close err: %s\n", err)
		} else {
			log.Printf("http close success\n")
		}
	})
	InsGlobalHook.Wait()
}
