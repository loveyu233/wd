# wd

`wd` 是一个面向 Go 服务的通用组件库，聚合了 HTTP 服务启动、Gin 中间件、JWT 认证、GORM/Redis 封装、Resty 客户端、Excel/加密/随机数等常用能力，帮助快速搭建可观测、可扩展的业务应用。

## 功能亮点
- **一体化 HTTP 启动器**（`http.go` + `gin_engine.go`）提供公共/私有路由注册、全局/认证中间件注入、健康检查、优雅关闭及运行参数（超时、前缀、日志等）的统一配置接口。
- **JWT 认证链路**（`auth_jwt.go`）具备登录/退出/刷新处理器、灵活的 Token 提取策略、Cookie 同步、可插拔 payload/授权钩子，并可自动接入 HTTP 启动器的私有路由。
- **数据访问与缓存**：`gorm.go` 暴露 `InitGormDB`、`GormDefaultLogger`，`redis.go` 封装 `redis.UniversalClient`、分布式锁（redsync）及多种配置项，便于以一致方式管理数据库和 Redis。
- **可观测中间件**：`middleware_trace_id.go`、`middleware_request_time.go`、`middleware_log.go` 等提供链路日志、TraceID、请求耗时、异常恢复等通用能力。
- **网络与工具集**：`resty.go` 提供默认 HTTP 客户端，`response.go`/`request.go` 统一请求/响应结构，`password.go`、`encrypt.go`、`random.go`、`snowflake.go` 等实现密码、加解密、随机 ID、ID 生成器。
- **场景扩展**：`pay/`、`login/`、`msg/`、`excel_*` 等子目录覆盖支付、登录、消息、Excel 处理等业务常见需求，可按需引用。

## 安装
```bash
go get github.com/loveyu233/wd@latest
```

## 快速开始
下面示例演示了如何：
1. 初始化 JWT 中间件；
2. 注册公开/私有路由；
3. 通过 `InitHTTPServerAndStart` 启动 HTTP 服务并自动接入默认中间件。

```go
package main

import (
    "log"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    wd "github.com/loveyu233/wd"
)

type account struct {
    Username string
}

type loginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

// accountPayload 定义 JWT Claims 的负载结构，字段名对应 JSON key。
type accountPayload struct {
    Username string `json:"username"`
    LoginAt  int64  `json:"login_at"`
}

var users = map[string]string{
    "alice": "123456",
    "bob":   "654321",
}

func main() {
    auth, err := wd.NewGinJWTMiddleware(
        // authenticator: 验证用户身份，返回具体类型
        func(c *gin.Context) (*account, error) {
            var req loginReq
            if err := c.ShouldBindJSON(&req); err != nil {
                return nil, err
            }
            if pwd, ok := users[req.Username]; !ok || pwd != req.Password {
                return nil, wd.MsgErrBadRequest("账号或密码错误")
            }
            return &account{Username: req.Username}, nil
        },
        // payloadFunc: 返回负载结构体，自动序列化为 JWT Claims
        func(data *account) accountPayload {
            return accountPayload{
                Username: data.Username,
                LoginAt:  time.Now().Unix(),
            }
        },
        // identityHandler: 直接使用结构体字段，无需从 map 中读取
        func(c *gin.Context, payload accountPayload) (any, error) {
            return payload.Username, nil
        },
        wd.WithJWTRealm("demo zone"),
        wd.WithJWTKey([]byte("change-me")),
        wd.WithJWTTimeout(30*time.Minute),
        wd.WithJWTMaxRefresh(time.Hour),
        wd.WithJWTIdentityKey("username"),
    )
    if err != nil {
        log.Fatalf("init jwt failed: %v", err)
    }

    wd.PublicRoutes.Append(func(rg *gin.RouterGroup) {
        rg.POST("/login", auth.LoginHandler())
    })
    wd.PrivateRoutes.Append(func(rg *gin.RouterGroup) {
        rg.GET("/profile", func(c *gin.Context) {
            claims := auth.ExtractClaims(c)
            c.JSON(http.StatusOK, gin.H{
                "claims": claims,
                "token":  wd.GetToken(c),
            })
        })
    })

    wd.InitHTTPServerAndStart(
        ":8080",
        wd.WithGinRouterPrefix("/api"),
        wd.WithGinRouterModel(wd.GinModelDebug),
    )
}
```

> `InitHTTPServerAndStart` 会阻塞当前 goroutine 并监听 SIGINT/SIGTERM；按 `Ctrl+C` 即可触发优雅关闭。

### 运行示例
仓库在 `test/jwt_demo` 中提供了更完整的演示，包括登录、刷新、写 Cookie 与 Admin 路由：
```bash
go run ./test/jwt_demo
```
使用 `curl` 验证：
```bash
curl -X POST http://127.0.0.1:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","password":"123456"}'
```
保留返回的 token 后访问受保护接口：
```bash
curl http://127.0.0.1:8080/api/profile \
  -H "Authorization: Bearer <TOKEN>"
```



 ## 目录速览
      .
      ├── auth_jwt.go            # JWT 中间件及登录/刷新/退出处理器
      ├── auth_jwt_options.go     # JWT 中间件函数选项（WithJWT* 系列）
      ├── gin_engine.go          # Gin Router 构建、公共/私有路由注册与中间件装配
      ├── http.go                # HTTP Server 启动器、优雅关闭、默认认证中间件挂载
      ├── middleware_trace_id.go # TraceID 注入
      ├── middleware_request_time.go # 请求耗时统计
      ├── middleware_log.go      # 链路日志与请求日志收集
      ├── middleware_recovery.go # panic 保护
      ├── middleware_cors.go     # CORS 配置中间件
      ├── gorm.go                # GORM 初始化、默认 Logger、全局 DB 实例
      ├── redis.go               # Redis/RedSync 封装与可配置初始化
      ├── resty.go               # Resty HTTP 客户端单例
      ├── response.go            # 统一响应结构及辅助函数
      ├── request.go             # 请求体解析/绑定工具
      ├── context.go             # 快捷 Context/超时工具
      ├── signal.go              # 系统信号 Hook，注册优雅关闭回调
      ├── random.go              # 随机字符串/数字工具
      ├── password.go            # 密码生成与校验
      ├── encrypt.go             # 加解密辅助
      ├── snowflake.go           # 雪花算法 ID 生成
      ├── template.go            # 模板渲染/文本处理工具
      ├── lo.go / str.go / time.go 等 # 常用数据/字符串/时间工具函数
      ├── excel_export.go        # Excel 导出封装
      ├── excel_mapper.go        # Excel 字段映射
      ├── excel_math.go          # Excel 统计计算辅助
      ├── msg/                   # 消息/公众号等集成（含 RegisterHandlers 框架）
      ├── login/                 # 登录渠道示例（如小程序）
      ├── pay/                   # 支付客户端与回调处理
      ├── go.mod / go.sum        # 依赖与版本
      └── README.md              # 项目说明（本次已更新）

## 贡献

欢迎通过 Issue/PR 提交新功能或修复，提交前请确保：
1. 新增/修改代码已通过 `go test ./...`；
2. 如引入新特性，请在 README 或相应示例中补充说明；
3. 遵循现有代码风格（使用 `gofmt`）。
