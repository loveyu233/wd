# WD 通用后台工具库

WD 是一套在 Go 语言项目中反复打磨的基础能力集合，主要面向 Web/微服务后台场景。仓库提供 HTTP 服务启动、Gin 中间件、配置加载、Redis/Gorm 客户端、Resty HTTP 客户端、任务调度、Excel 导出、加解密、支付/消息等集成能力，帮助团队快速搭建具备观察性和业务扩展点的服务。

## 功能特性
- **HTTP 服务脚手架**：`http.go` + `gin_engine.go` 封装了 Gin 引擎注册、公开/私有路由拆分、启动优雅关闭、前缀/模式/超时/日志参数化配置。
- **标准化中间件与响应**：`middleware_*.go`、`response.go` 提供 TraceID、耗时统计、Recover、访问日志、JSON 响应、错误码/翻译等通用链路能力。
- **基础设施客户端包装**：`redis.go`、`gorm.go`、`resty.go`、`ants.go`、`context.go`、`auth_jwt.go` 等实现 Redis/Gorm 初始化、Resty 单例、协程池、Context 与 JWT 认证中间件等高频需求。
- **数据处理与工具集**：`excel_export.go`、`excel_mapper.go`、`excel_math.go`、`encrypt.go`、`decimal.go`、`diff.go`、`str.go`、`time.go` 等聚合了 Excel 导入/导出、加解密、精准计算、Diff、字符串/时间工具。
- **任务与异步能力**：`corn.go` 结合 gocron + Redis 锁实现多实例唯一调度、任务生命周期回调；`hook`/`signal.go` 协助进程优雅退出。
- **业务集成示例**：`login/`、`msg/`、`pay/` 目录包含微信小程序登录、公众号消息、微信/支付宝支付的处理示例，可作为二次封装模板。

## 环境要求
- Go 1.25 及以上版本（参考 `go.mod`）
- 可选依赖：MySQL、Redis、gocron、Resty 等根据具体模块选择使用

## 安装
```bash
go get github.com/loveyu233/wd
```

## 快速开始
以下示例展示如何使用内置 Gin 启动器暴露公开路由，并复用默认中间件链：

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/loveyu233/wd"
)

func init() {
    wd.PublicRoutes = append(wd.PublicRoutes, func(r *gin.RouterGroup) {
        r.GET("/ping", func(c *gin.Context) {
            wd.ResponseSuccess(c, gin.H{"message": "pong"})
        })
    })
}

func main() {
    wd.InitHTTPServerAndStart(
        ":8080",
        wd.WithGinRouterPrefix("/api"),
        wd.WithGinRouterModel(wd.GinModelRelease),
    )

    // 服务在独立 goroutine 中启动，如果 main 函数直接返回可根据需要阻塞
    select {}
}
```

启动后访问 `http://localhost:8080/api/ping` 即可得到统一格式的 JSON 响应，并自动注入 TraceID、访问日志与异常恢复链路。

## 常见用法示例
### Redis 客户端
示例依赖 `time` 包：
```go
import "time"

if err := wd.InitRedis(
    wd.WithRedisAddressOption([]string{"127.0.0.1:6379"}),
    wd.WithRedisPasswordOption("example"),
    wd.WithRedisDBOption(0),
); err != nil {
    panic(err)
}

key := "captcha:login:123456"
_ = wd.InsRedis.SetCaptcha(key, "0426", 5*time.Minute)
code, err := wd.InsRedis.GetCaptcha(key)
```

### Resty HTTP 客户端
`resty.go` 通过 `sync.Once` 提供并发安全的 Resty 单例，可方便地注入统一超时/重试策略。
```go
import "fmt"

resp, err := wd.RestyClient().R().
    SetHeader("X-Trace-Id", wd.NewTraceID()).
    SetBody(map[string]any{"foo": "bar"}).
    Post("https://httpbin.org/post")
if err != nil {
    panic(err)
}
fmt.Println(resp.Status())
```

### 定时任务
示例依赖 `context`、`fmt`、`time`、`github.com/go-co-op/gocron/v2` 以及 `github.com/google/uuid`：
```go
if err := wd.InitCornJob(
    wd.WithBeforeJobRuns(func(id uuid.UUID, name string) { fmt.Println("before", name) }),
); err != nil {
    panic(err)
}

wd.InsCornJob.RunJobEveryDuration(5*time.Minute, gocron.NewTask(func(ctx context.Context) {
    fmt.Println("run job")
}))
```

### Excel 导出
```go
exporter := wd.InitExcelExporter(
    wd.WithExcelExporterSheetName("用户"),
)
err := exporter.ExportToFile(users, "users.xlsx")
```

更多函数可以在对应文件查看注释，所有公共方法都包含中文说明，便于通过 `GoDoc` 或源码直接阅读。

## 目录速览
- `middleware_*.go`：TraceID、日志、跨域、请求耗时、异常恢复等 Gin 中间件实现。
- `http.go` / `gin_engine.go`：服务启动、路由注册、优雅退出逻辑。
- `redis.go` / `gorm.go` / `resty.go`：基础设施客户端封装与初始化。
- `request.go` / `response.go`：请求解析、参数校验、统一响应结构与错误码。
- `excel_*.go`：Excel 模板、导出、数据映射与数学函数集合。
- `login/`、`msg/`、`pay/`：对接微信/支付宝的示例服务。
- `time.go` / `context.go` / `id.go` / `snowflake.go`：时间、上下文、ID 生成等工具。

## 贡献指南
欢迎提交 Issue / PR：
1. Fork 本仓库并创建特性分支。
2. 补充或更新相应的单元测试/示例。
3. 提交前运行必要的格式化与静态检查。

## 许可证
开源协议请参考仓库中随附的 License 文件（如无特别说明则以该文件为准）。
