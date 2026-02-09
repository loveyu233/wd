package wd

import (
	"io"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	PublicRoutes      PublicRoutesType  // 存储无需认证的公开路由处理函数
	PrivateRoutes     PrivateRoutesType // 存储需要认证的私有路由处理函数
	publicRoutesLock  sync.Mutex
	privateRoutesLock sync.Mutex
)

type PublicRoutesType []func(*gin.RouterGroup)
type PrivateRoutesType []func(*gin.RouterGroup)

func (pr *PublicRoutesType) Append(f func(*gin.RouterGroup)) {
	publicRoutesLock.Lock()
	defer publicRoutesLock.Unlock()
	*pr = append(*pr, f)
}

func (pr *PrivateRoutesType) Append(f func(*gin.RouterGroup)) {
	privateRoutesLock.Lock()
	defer privateRoutesLock.Unlock()
	*pr = append(*pr, f)
}

type RouterConfig struct {
	outputHealthz    bool              // 是否输出健康检查请求的日志输出
	model            GinModel          // gin启动模式
	prefix           string            // api前缀
	authMiddleware   []gin.HandlerFunc // 认证api的中间件
	globalMiddleware []gin.HandlerFunc // 全局中间件
	recordHeaderKeys []string          // 需要记录的请求头
	saveLog          func(ReqLog)      // 保存请求日志
	readTimeout      time.Duration
	writeTimeout     time.Duration
	idleTimeout      time.Duration
	maxHeaderBytes   int
	skipLog          bool
	logWriter        io.Writer
	contentKeys      []string
	engineFunc       func(engine *gin.Engine)
}

type GinModel string

// String 用来返回 GinModel 的字符串表现形式。
func (m GinModel) String() string {
	return string(m)
}

var (
	GinModelRelease GinModel = "release"
	GinModelDebug   GinModel = "debug"
	GinModelTest    GinModel = "test"
)

type GinRouterConfigOption func(*RouterConfig)

// WithGinSkipLog 用来控制是否跳过访问日志中间件。
func WithGinSkipLog(skipLog bool) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.skipLog = skipLog
	}
}

func WithGinEngineFunc(fun func(engine *gin.Engine)) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.engineFunc = fun
	}
}

func WithLogWriter(w io.Writer) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.logWriter = w
	}
}
func WithLogContentKeys(keys []string) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.contentKeys = keys
	}
}

// WithGinReadTimeout 用来设置 HTTP 服务器的读取超时。
func WithGinReadTimeout(d time.Duration) GinRouterConfigOption {
	return func(routerConfig *RouterConfig) {
		routerConfig.readTimeout = d
	}
}

// WithGinWriteTimeout 用来设置 HTTP 响应写入超时。
func WithGinWriteTimeout(d time.Duration) GinRouterConfigOption {
	return func(routerConfig *RouterConfig) {
		routerConfig.writeTimeout = d
	}
}

// WithGinIdleTimeout 用来设置连接空闲超时时间。
func WithGinIdleTimeout(d time.Duration) GinRouterConfigOption {
	return func(routerConfig *RouterConfig) {
		routerConfig.idleTimeout = d
	}
}

// WithGinMaxHeaderBytes 用来限制请求头允许的最大字节数。
func WithGinMaxHeaderBytes(d int) GinRouterConfigOption {
	return func(routerConfig *RouterConfig) {
		routerConfig.maxHeaderBytes = d
	}
}

// WithGinRouterModel 用来指定 gin 运行模式。
func WithGinRouterModel(model GinModel) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.model = model
	}
}

// WithGinRouterOutputHealthzLog 用来允许 healthz 请求输出日志。
func WithGinRouterOutputHealthzLog() GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.outputHealthz = true
	}
}

// WithGinRouterPrefix 用来设置 API 前缀。
func WithGinRouterPrefix(prefix string) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.prefix = prefix
	}
}

// WithGinRouterAuthHandler 用来配置需要鉴权的中间件。
func WithGinRouterAuthHandler(handlers ...gin.HandlerFunc) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.authMiddleware = handlers
	}
}

// WithGinRouterGlobalMiddleware 用来注册全局中间件链。
func WithGinRouterGlobalMiddleware(handlers ...gin.HandlerFunc) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.globalMiddleware = handlers
	}
}

// WithGinRouterLogRecordHeaderKeys 用来指定需要记录的请求头。
func WithGinRouterLogRecordHeaderKeys(keys []string) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.recordHeaderKeys = keys
	}
}

// WithGinRouterLogSaveLog 用来注入持久化请求日志的回调。
func WithGinRouterLogSaveLog(f func(ReqLog)) GinRouterConfigOption {
	return func(config *RouterConfig) {
		config.saveLog = f
	}
}

// initPrivateRouter 用来组装带公共和私有路由的 gin 引擎。
func initPrivateRouter(config RouterConfig) *gin.Engine {
	publicRoutes := make([]func(*gin.RouterGroup), 0, len(PublicRoutes)+1)
	publicRoutes = append(publicRoutes, func(group *gin.RouterGroup) {
		if !config.outputHealthz {
			group.GET("/healthz", GinLogSetSkipLogFlag(), func(c *gin.Context) {
				c.Status(200)
			})
		} else {
			group.GET("/healthz", func(c *gin.Context) {
				c.Status(200)
			})
		}
	})
	publicRoutes = append(publicRoutes, PublicRoutes...)

	// 复制 PrivateRoutes 避免修改原始切片
	privateRoutes := make([]func(*gin.RouterGroup), len(PrivateRoutes))
	copy(privateRoutes, PrivateRoutes)

	config.globalMiddleware = append(config.globalMiddleware, MiddlewareTraceID(), MiddlewareRequestTime(), MiddlewareRecovery())
	if !config.skipLog {
		config.globalMiddleware = append(config.globalMiddleware, MiddlewareLogger(MiddlewareLogConfig{
			HeaderKeys:  config.recordHeaderKeys,
			SaveLog:     config.saveLog,
			LogWriter:   config.logWriter,
			ContentKeys: config.contentKeys,
		}))
	}

	engine := newGinRouter(config.model, config.globalMiddleware...)
	registerRoutes(engine, config.prefix, publicRoutes, privateRoutes, config.authMiddleware...)
	return engine
}

// newGinRouter 用来创建指定模式的 gin.Engine 并挂载中间件。
func newGinRouter(mode GinModel, globalMiddlewares ...gin.HandlerFunc) *gin.Engine {
	gin.SetMode(mode.String())
	engine := gin.New()

	// 添加中间件
	engine.Use(globalMiddlewares...)

	return engine
}

// registerRoutes 用来在基本路径下注入公开和私有路由。
func registerRoutes(r *gin.Engine, baseRouterPrefix string, publicRoutes, privateRoutes []func(*gin.RouterGroup), authMiddlewares ...gin.HandlerFunc) {
	baseRouter := r.Group(baseRouterPrefix)
	for _, route := range publicRoutes {
		route(baseRouter)
	}

	priRoute := baseRouter.Group("", authMiddlewares...)
	for _, route := range privateRoutes {
		route(priRoute)
	}
}
