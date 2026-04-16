# wd

`wd` 是一个偏服务端场景的 Go 工具库，核心目标不是只提供几个零散函数，而是把一个业务项目里最常反复写的能力收拢成一套统一工具：

- HTTP 服务启动与 Gin 路由组织
- JWT 认证与 Claims 提取
- 统一响应、参数校验、PATCH 三态字段
- GORM 初始化、SQL 日志增强、gorm/gen 代码生成辅助
- Redis 初始化、Lua 工具脚本、分布式锁、限流、排行榜、库存等场景能力
- 阿里云短信发送能力
- Excel 导入导出
- 时间类型、文件上传、加密、模板、脱敏、随机串等通用工具

这份 README 不再按“文件逐个解释源码”，而是按“当前项目里有哪些工具、分别解决什么问题、怎么接入”来组织。你可以把它当成这个库的使用手册。

## 适合什么项目

`wd` 适合以下类型的 Go 服务项目：

- 使用 `gin` 做 HTTP 接口层
- 使用 `gorm` / `gorm/gen` 做数据库访问
- 需要统一错误码、日志、TraceID、响应结构
- 需要短信发送等轻量业务能力接入
- 需要大量“工具层代码”但不想每个项目都重复造轮子

## 模块地图

| 模块 | 主要文件/包 | 解决的问题 | 主入口 |
| --- | --- | --- | --- |
| HTTP 启动 | `http_server.go` `gin_engine.go` | 启动 Gin 服务、组织公开/私有路由、优雅关闭 | `InitHTTPServerAndStart` `NewHTTPServer` |
| 请求链路日志 | `middleware_log.go` `middleware_trace_id.go` `middleware_request_time.go` `middleware_recovery.go` | TraceID、请求耗时、统一日志、阶段耗时、异常恢复 | `MiddlewareLogger` `BeginStageTiming` |
| JWT 认证 | `auth_jwt.go` `auth_jwt_options.go` | 登录、鉴权、刷新、Claims 提取、Cookie/RSA 支持 | `NewGinJWTMiddleware` |
| 响应与错误 | `response.go` `params_verify.go` `gin_param.go` | 统一响应体、错误码、中文参数校验、query/path 参数读取 | `ResponseSuccess` `ResponseError` |
| PATCH/查询参数 | `patch_field.go` `patch_field_assign.go` `params_precompiled.go` | PATCH 三态字段、分页、范围查询、文件表单辅助 | `Field[T]` `PatchUpdate` `ReqRange` |
| GORM 工具 | `gorm.go` `gen.go` `gen_field.go` | 初始化 DB、增强 SQL 日志、gorm/gen 代码生成 | `InitGormDB` `InsDB.Gen` |
| Redis 工具 | `redis.go` `redis_lua.go` | 初始化 Redis、分布式锁、限流、排行榜、库存、Bloom、ID 生成 | `InitRedis` |
| 定时/权限/搜索 | `cron_task.go` `casbin.go` `es.go` | 分布式定时任务、RBAC、Elasticsearch 批量写入 | `InitCronJob` `InitCasbin` `InitEs` |
| 短信服务 | `sms.go` | 阿里云短信发送能力 | `NewSMS` `NewSMSWithAccessKey` `NewSMSWithClient` |
| Excel 工具 | `excel_export.go` `excel_mapper.go` `excel_math.go` | Excel 导出、导入、坐标换算 | `InitExcelExporter` `InitExcelMapper` |
| 时间与 SQL 类型 | `sql_type.go` `time.go` | `DateTime`/`DateOnly`/`TimeOnly`/`TimeHM` 类型与时间工具 | `Now` `ParseDateTimeValue` |
| 通用工具 | `file.go` `resty.go` `encrypt.go` `random.go` 等 | 文件上传、HTTP 调用、加密、脱敏、模板、Diff 等 | 各文件导出函数 |

## 专题文档

如果你是按业务能力来接入，而不是按源码文件来查，建议直接看拆分后的专题文档：

- `docs/README.md`：文档导航
- `docs/message.md`：阿里云短信接入

## 目录导航

- [专题文档](#专题文档)
- [安装](#安装)
- [5 分钟快速接入](#5-分钟快速接入)
- [1. HTTP 服务与 Gin 工具](#1-http-服务与-gin-工具)
- [2. 请求日志、TraceID 与阶段耗时](#2-请求日志traceid-与阶段耗时)
- [3. JWT 认证工具](#3-jwt-认证工具)
- [4. 统一响应、错误与参数校验](#4-统一响应错误与参数校验)
- [5. PATCH 三态字段、分页、范围查询、文件参数](#5-patch-三态字段分页范围查询文件参数)
- [6. GORM 初始化、SQL 日志与 gorm/gen 工具](#6-gorm-初始化sql-日志与-gormgen-工具)
- [7. 时间类型与时间工具](#7-时间类型与时间工具)
- [8. Redis 初始化、分布式锁与 Lua 工具集](#8-redis-初始化分布式锁与-lua-工具集)
- [9. 定时任务、Casbin 权限与 Elasticsearch](#9-定时任务casbin-权限与-elasticsearch)
- [10. 短信服务 `sms.go`](#10-短信服务-smsgo)
- [11. Excel 导入导出工具](#11-excel-导入导出工具)
- [12. 文件、HTTP、加密、随机串等常用工具](#12-文件http加密随机串等常用工具)
- [附录：按文件查 API](#附录按文件查-api)
- [13. 其他基础工具索引](#13-其他基础工具索引)
- [14. 推荐阅读顺序](#14-推荐阅读顺序)
- [15. 仓库内示例与测试](#15-仓库内示例与测试)
- [16. 总结](#16-总结)

## 安装

```bash
go get github.com/loveyu233/wd@latest
```

## 5 分钟快速接入

下面这个例子串起来了最常见的接入方式：

1. 创建 JWT 中间件；
2. 注册公开/私有路由；
3. 启动 HTTP 服务；
4. 让私有路由自动走鉴权。

```go
package main

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    wd "github.com/loveyu233/wd"
)

type loginReq struct {
    Username string `json:"username" binding:"required"`
    Password string `json:"password" binding:"required"`
}

type user struct {
    ID       uint64 `json:"id"`
    Username string `json:"username"`
}

type claims struct {
    UserID   uint64 `json:"user_id"`
    Username string `json:"username"`
}

func main() {
    authMW, err := wd.NewGinJWTMiddleware(
        func(c *gin.Context) (*user, error) {
            var req loginReq
            if err := c.ShouldBindJSON(&req); err != nil {
                return nil, err
            }
            if req.Username != "demo" || req.Password != "123456" {
                return nil, wd.MsgErrBadRequest("账号或密码错误")
            }
            return &user{ID: 1, Username: req.Username}, nil
        },
        func(data *user) claims {
            return claims{UserID: data.ID, Username: data.Username}
        },
        func(c *gin.Context, payload claims) (any, error) {
            return payload.UserID, nil
        },
        wd.WithJWTRealm("demo"),
        wd.WithJWTKey([]byte("replace-with-your-secret")),
        wd.WithJWTTimeout(2*time.Hour),
        wd.WithJWTIdentityKey("user_id"),
    )
    if err != nil {
        panic(err)
    }

    wd.PublicRoutes.Append(func(rg *gin.RouterGroup) {
        rg.POST("/login", authMW.LoginHandler())
        rg.GET("/ping", func(c *gin.Context) {
            wd.ResponseSuccess(c, gin.H{"pong": true})
        })
    })

    wd.PrivateRoutes.Append(func(rg *gin.RouterGroup) {
        rg.GET("/profile", func(c *gin.Context) {
            payload, err := wd.ExtractClaimsAs[claims](c)
            if err != nil {
                wd.ResponseError(c, err)
                return
            }
            c.JSON(http.StatusOK, gin.H{
                "code": 200,
                "data": payload,
            })
        })
    })

    wd.InitHTTPServerAndStart(
        ":8080",
        wd.WithGinRouterPrefix("/api"),
        wd.WithGinRouterAuthHandler(authMW.MiddlewareFunc()),
        wd.WithGinRouterModel(wd.GinModelDebug),
    )
}
```

启动后：

- 公开接口：`POST /api/login`、`GET /api/ping`
- 私有接口：`GET /api/profile`
- 健康检查：`GET /api/healthz`

---

## 1. HTTP 服务与 Gin 工具

### 1.1 这个模块做什么

`http_server.go` + `gin_engine.go` 负责把 Gin 服务的常规样板代码统一掉：

- 公开路由和私有路由分开注册
- 自动挂载 TraceID、请求耗时、Recovery、请求日志中间件
- 支持 API 前缀、超时、header 限制、全局中间件、鉴权中间件
- 支持优雅关闭

### 1.2 主要入口

- `wd.PublicRoutes.Append(func(*gin.RouterGroup))`
- `wd.PrivateRoutes.Append(func(*gin.RouterGroup))`
- `wd.InitHTTPServerAndStart(addr, opts...)`
- `wd.NewHTTPServer(addr, opts...)`

### 1.3 常用选项

- `wd.WithGinRouterPrefix("/api")`：统一前缀
- `wd.WithGinRouterAuthHandler(authMW.MiddlewareFunc())`：私有路由鉴权链
- `wd.WithGinRouterGlobalMiddleware(...)`：追加全局中间件
- `wd.WithGinRouterModel(wd.GinModelDebug)`：切换 Gin 模式
- `wd.WithGinReadTimeout(...)` / `wd.WithGinWriteTimeout(...)`
- `wd.WithGinRouterLogRecordHeaderKeys([]string{"X-Request-Id"})`
- `wd.WithGinRouterLogSaveLog(func(wd.ReqLog){ ... })`

### 1.4 什么时候用 `PublicRoutes` / `PrivateRoutes`

- `PublicRoutes`：无需登录即可访问的接口，如登录、回调、健康检查
- `PrivateRoutes`：必须在 `WithGinRouterAuthHandler(...)` 下访问的接口

示例：

```go
wd.PublicRoutes.Append(func(rg *gin.RouterGroup) {
    rg.POST("/login", authMW.LoginHandler())
    rg.POST("/callback", callbackHandler)
})

wd.PrivateRoutes.Append(func(rg *gin.RouterGroup) {
    rg.GET("/me", profileHandler)
    rg.GET("/orders", orderListHandler)
})
```

---

## 2. 请求日志、TraceID 与阶段耗时

### 2.1 默认会自动挂什么

通过 `InitHTTPServerAndStart` / `NewHTTPServer` 初始化服务时，会自动挂上：

- `MiddlewareTraceID()`
- `MiddlewareRequestTime()`
- `MiddlewareRecovery()`
- `MiddlewareLogger(...)`（除非显式 `WithGinSkipLog(true)`）

### 2.2 日志能力

`middleware_log.go` 不是简单打印一行 access log，而是做了“请求级日志聚合”：

- 同一次请求内的业务日志先进入缓冲区
- 请求结束时一次性输出
- 可以带结构化字段、对象 payload、SQL 日志、阶段耗时
- 可以自定义快照模式，避免日志被后续对象修改污染

### 2.3 常用 API

- `wd.WriteGinInfoLog(c, key, format, args...)`
- `wd.WriteGinWarnLog(c, key, format, args...)`
- `wd.WriteGinErrAnyLog(c, key, payload)`
- `wd.GinLogSetModuleName("订单模块")`
- `wd.GinLogSetOptionName("创建订单")`
- `wd.BeginStageTiming(c, "查询数据库")`
- `wd.GetTraceID(c)`

### 2.4 记录阶段耗时示例

```go
stage := wd.BeginStageTiming(c, "查询订单")
order, err := query.Order.WithContext(c).Where(query.Order.ID.Eq(id)).First()
stage.Commit()
if err != nil {
    wd.ResponseError(c, err)
    return
}
wd.WriteGinInfoAnyLog(c, "order_result", order)
wd.ResponseSuccess(c, order)
```

---

## 3. JWT 认证工具

### 3.1 适合什么场景

`auth_jwt.go` 适合两类场景：

- 传统用户名密码登录 -> JWT
- 第三方登录成功后，复用 `TokenGenerator` 发业务 token

### 3.2 核心能力

- 登录：`LoginHandler()`
- 鉴权：`MiddlewareFunc()`
- 刷新：`RefreshHandler()` / `RefreshToken()`
- 手动发 token：`TokenGenerator(data)`
- Claims 提取：`ExtractClaimsAs[T](c)`
- 身份读取：`GetIdentityAs[T](mw, c)`
- 原始 token 读取：`GetToken(c)`
- 支持 Header / Query / Cookie / Param / Form 多来源取 token
- 支持 RSA、公私钥、Cookie 同步写入

### 3.3 最常用配置项

- `wd.WithJWTKey([]byte("secret"))`
- `wd.WithJWTTimeout(2*time.Hour)`
- `wd.WithJWTMaxRefresh(24*time.Hour)`
- `wd.WithJWTIdentityKey("user_id")`
- `wd.WithJWTTokenLookup("header:Authorization,cookie:jwt")`
- `wd.WithJWTCookie(...)`
- `wd.WithJWTRSA(...)`

### 3.4 第三方登录后手动签发 token

这个模式在第三方登录或外部认证成功后手动发 token 的场景里很常见：

```go
jwtMW, _ := wd.NewGinJWTMiddleware(
    func(c *gin.Context) (*user, error) {
        return nil, wd.MsgErrBadRequest("当前示例不走 LoginHandler")
    },
    func(data *user) claims {
        return claims{UserID: data.ID, Username: data.Username}
    },
    func(c *gin.Context, payload claims) (any, error) {
        return payload.UserID, nil
    },
    wd.WithJWTKey([]byte("secret")),
)

token, expire, err := jwtMW.TokenGenerator(&user{ID: 1, Username: "demo"})
_ = expire
_ = token
_ = err
```

---

## 4. 统一响应、错误与参数校验

### 4.1 响应结构

`response.go` 统一响应格式：

```json
{
  "code": 200,
  "message": "操作成功",
  "data": {}
}
```

### 4.2 常用函数

- `wd.ResponseSuccess(c, data)`
- `wd.ResponseSuccessMsg(c, "保存成功")`
- `wd.ResponseSuccessToken(c, token)`
- `wd.ResponseSuccessEncryptData(c, data, customKeyFunc)`
- `wd.ResponseError(c, err)`
- `wd.ResponseParamError(c, err)`

### 4.3 AppError 体系

`wd` 内置了一套业务错误码：

- 外部服务失败：`MsgErrRequestWechat` / `MsgErrRequestWechatPay` 等
- 参数错误：`MsgErrInvalidParam`
- 未登录：`MsgErrUnauthorized`
- 权限不足：`MsgErrForbiddenAuth`
- 数据不存在：`MsgErrNotFound`
- 冲突：`MsgErrUniqueIndexConflict` / `MsgErrVERSION_CONFLICT`
- 服务错误：`MsgErrServerBusy` / `MsgErrDatabase` / `MsgErrRedis`

如果你已经有统一错误治理，这套错误码可以直接拿来当项目默认规范。

### 4.4 参数辅助

- `wd.GinQueryDefault[T](c, key, defaultValue)`
- `wd.GinQueryRequired[T](c, key)`
- `wd.GinPathRequired[T](c, key)`
- `wd.TranslateError(err)`：把 Gin / validator / JSON 解析错误翻成中文

示例：

```go
orderID, err := wd.GinPathRequired[uint64](c, "id")
if err != nil {
    wd.ResponseParamError(c, err)
    return
}
```

---

## 5. PATCH 三态字段、分页、范围查询、文件参数

### 5.1 `Field[T]`：解决 PATCH 场景最难处理的问题

普通结构体字段无法区分三件事：

1. 前端没传；
2. 前端显式传了 `null`；
3. 前端传了新值。

`wd.Field[T]` 专门解决这个问题。

```go
type UpdateUserReq struct {
    Nickname wd.Field[string]      `json:"nickname" binding:"min=2,max=8"`
    Birthday wd.Field[wd.DateOnly] `json:"birthday"`
}
```

核心字段：

- `Set`：这个字段有没有在请求里出现
- `Null`：是不是显式传了 `null`
- `Value`：实际值

### 5.2 `PatchUpdate`：把 PATCH 请求直接转成 gorm/gen 赋值表达式

```go
updates := []field.AssignExpr{}

assignExpr, changed, err := wd.PatchUpdate(req.Nickname, oldUser.Nickname, query.User.Nickname)
if err != nil {
    wd.ResponseError(c, err)
    return
}
if changed {
    updates = append(updates, assignExpr)
}
```

适合配合 `gorm/gen` 的 `UpdateColumnSimple(...)` 使用。

### 5.2.1 `BuildGenUpdates`：一键生成整组 `gorm/gen` 更新表达式

如果请求结构里的 PATCH 字段较多，可以直接一次性生成更新表达式：

```go
type UpdateUserReq struct {
    NickName wd.Field[string]      `json:"nickname" patch:"Nickname"`
    Email    wd.Field[string]      `json:"email"`
    Birthday wd.Field[wd.DateOnly] `json:"birthday" patch:"Date"`
}

updates, err := wd.BuildGenUpdates(req, oldUser, query.User)
if err != nil {
    wd.ResponseError(c, err)
    return
}

if len(updates) == 0 {
    wd.ResponseSuccess(c, "无修改内容")
    return
}

_, err = query.User.WithContext(c).
    Where(query.User.ID.Eq(userID)).
    UpdateColumnSimple(updates...)
```

规则说明：

- 默认按请求字段名去匹配 `oldModel` 和 `query.User` 的字段名
- 可以用 `patch:"Nickname"` 这类标签显式指定字段名
- `oldModel` 允许传 `nil`，此时只要请求中显式传了字段，就会直接生成更新表达式
- `patch:"-"` 表示跳过该字段，不参与自动构建

完整示例：

```go
type UpdateUserReq struct {
    Version  uint64                `json:"version" binding:"required"`
    Nickname wd.Field[string]      `json:"nickname"`
    Email    wd.Field[string]      `json:"email"`
    Birthday wd.Field[wd.DateOnly] `json:"birthday" patch:"Date"`
    Tags     wd.Field[[]string]    `json:"tags"`
}

group.PATCH("/user/:id", func(c *gin.Context) {
    var req UpdateUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        wd.ResponseParamError(c, err)
        return
    }

    userID, err := wd.GinPathRequired[uint64](c, "id")
    if err != nil {
        wd.ResponseParamError(c, err)
        return
    }

    oldUser, err := query.User.WithContext(c).
        Where(query.User.ID.Eq(userID)).
        First()
    if err != nil {
        wd.ResponseError(c, err)
        return
    }

    if oldUser.Version != req.Version {
        wd.ResponseError(c, wd.MsgErrVERSION_CONFLICT("当前为旧数据"))
        return
    }

    updates, err := wd.BuildGenUpdates(req, oldUser, query.User)
    if err != nil {
        wd.ResponseError(c, err)
        return
    }

    if len(updates) == 0 {
        wd.ResponseSuccess(c, "无修改内容")
        return
    }

    updates = append(updates, query.User.Version.Add(1))

    _, err = query.User.WithContext(c).
        Where(query.User.ID.Eq(userID), query.User.Version.Eq(req.Version)).
        UpdateColumnSimple(updates...)
    if err != nil {
        wd.ResponseError(c, err)
        return
    }

    wd.ResponseSuccess(c, "ok")
})
```

如果不需要先查旧值，也可以直接传 `nil`：

```go
updates, err := wd.BuildGenUpdates(req, nil, query.User)
if err != nil {
    wd.ResponseError(c, err)
    return
}
```

此时的行为是：

- 没传字段：跳过
- 显式传普通值：直接生成 `Value(...)`
- 显式传 `null`：如果目标字段支持 `Null()`，则生成置空更新

常见标签写法：

```go
type UpdateUserReq struct {
    Name   wd.Field[string] `json:"name" patch:"Nickname"`
    Remark wd.Field[string] `json:"remark" patch:"-"`
}
```

- `patch:"Nickname"`：把请求字段映射到 `query.User.Nickname`
- `patch:"-"`：跳过该字段，不参与自动构建

### 5.3 预编译请求结构

`params_precompiled.go` 提供了一组非常实用的请求结构：

- `wd.ReqRange[T]`：时间范围
- `wd.ReqKeyword`：关键词模糊查询
- `wd.ReqPageSize`：分页
- `wd.ReqFile`：单文件上传表单
- `wd.ReqFiles`：多文件上传表单
- `wd.ApplyPage(page, query.User)`：把分页应用到 `gorm/gen` 查询对象

时间范围示例：

```go
type ListReq struct {
    wd.ReqRange[wd.DateTime]
    wd.ReqPageSize
}

var req ListReq
if err := c.ShouldBindQuery(&req); err != nil {
    wd.ResponseParamError(c, err)
    return
}

expr, err := req.ReqRange.WhereExpr(query.Order, query.Order.CreatedAt)
if err != nil {
    wd.ResponseError(c, err)
    return
}

list, err := wd.ApplyPage(req.ReqPageSize, query.Order).
    Where(expr).
    Find()
```

文件表单示例：

```go
type UploadReq struct {
    wd.ReqFile
}

var req UploadReq
if err := c.ShouldBind(&req); err != nil {
    wd.ResponseParamError(c, err)
    return
}

contentType, _ := req.ContentType()
size, _ := req.Size()
wd.ResponseSuccess(c, gin.H{"content_type": contentType, "size": size})
```

---

## 6. GORM 初始化、SQL 日志与 gorm/gen 工具

### 6.1 初始化数据库

入口是 `wd.InitGormDB(...)`。

```go
err := wd.InitGormDB(
    wd.GormConnConfig{
        Username: "root",
        Password: "",
        Host:     "127.0.0.1",
        Port:     3306,
        Database: "demo",
        Params: map[string]any{
            "charset":   "utf8mb4",
            "parseTime": true,
            "loc":       "Asia/Shanghai",
        },
    },
    wd.GormDefaultLogger(
        wd.WithGormConfigLogLevel(4),
        wd.WithGormConfigCallerPathMode(wd.GormCallerPathModeModuleRelative),
    ),
)
if err != nil {
    panic(err)
}
```

初始化完成后，全局连接在 `wd.InsDB`。

### 6.2 GORM 日志增强

`gorm.go` 做了两件很实用的事情：

- 给 GORM SQL 日志补上调用方文件定位
- 支持把 SQL 日志挂入请求级日志缓冲区

可用工具：

- `wd.GormDefaultLogger(...)`
- `wd.WrapGormLoggerWithRequestLogger(base)`

### 6.3 运行 gorm/gen

入口是 `wd.InsDB.Gen(...)`。

```go
wd.InsDB.Gen(
    wd.WithGenOutFilePath("test/httpt/gen/query"),
    wd.WithGenUseTablesName("user", "audit_log"),
    wd.WithGenGlobalColumnTypeAddDatatypes(),
    wd.WithGenTableColumnType(map[string][]wd.GenFieldType{
        "user": {
            {
                ColumnName: "profile",
                ColumnType: "datatypes.JSONMap",
                IsJsonStatusType: true,
            },
        },
    }),
)
```

### 6.4 `gen_field.go`：生成查询表达式的小工具

常用函数：

- `wd.GenJSONArrayQuery(column)`
- `wd.GenJSONArrayQueryContainsValue(column, value)`
- `wd.GenCustomTimeBetween(table, column, left, right)`
- `wd.GenNewBetween(table, column, left, right)`
- `wd.GenNewUnsafeFieldRaw(rawSQL, vars...)`

如果你的项目已经大量使用 `gorm/gen`，这些函数能显著减少重复拼条件代码。

---

## 7. 时间类型与时间工具

### 7.1 四个自定义时间类型

`sql_type.go` + `time.go` 提供了四个可直接用于：

- JSON 序列化
- Gin 参数绑定
- GORM Scan / Value
- 业务计算

的类型：

- `wd.DateTime`：`YYYY-MM-DD HH:MM:SS`
- `wd.DateOnly`：`YYYY-MM-DD`
- `wd.TimeOnly`：`HH:MM:SS`
- `wd.TimeHM`：`HH:MM`

示例：

```go
type User struct {
    Birthday wd.DateOnly `json:"birthday"`
    StartAt  wd.DateTime `json:"start_at"`
}
```

### 7.2 常用时间函数

- `wd.Now()`
- `wd.NowAsDateTime()` / `wd.NowAsDateOnly()`
- `wd.ParseDateTimeValue(str)`
- `wd.NewDateOnlyString("2026-04-15")`
- `wd.NewTimeHMString("18:30")`
- `wd.FormatDateTime(t)`
- `wd.TodayRange()` / `wd.YesterdayRange()`
- `wd.CurrentMonthRange()` / `wd.LastMonthRange()`
- `wd.HasTimeConflict(...)`

### 7.3 典型场景

- 数据库模型字段直接用 `wd.DateOnly` / `wd.DateTime`
- 查询参数里直接绑定 `ReqRange[wd.DateOnly]`
- 排班、营业时间、预约时间冲突判断用 `TimeRange`

---

## 8. Redis 初始化、分布式锁与 Lua 工具集

### 8.1 初始化 Redis

```go
err := wd.InitRedis(
    wd.WithRedisAddressOption([]string{"127.0.0.1:6379"}),
    wd.WithRedisDBOption(0),
    wd.WithRedisPasswordOption(""),
)
if err != nil {
    panic(err)
}
```

初始化完成后，全局客户端在 `wd.InsRedis`。

### 8.2 常用基础能力

- `wd.InsRedis.NewLock(key)`：基于 redsync 的分布式锁
- `wd.InsRedis.SetCaptcha(key, value, ttl)`
- `wd.InsRedis.GetCaptcha(key)`
- `wd.InsRedis.DelCaptcha(key)`
- `wd.InsRedis.FindAllBitMapByTargetValue(key, targetValue)`

### 8.3 Lua 脚本工具覆盖的场景

`redis_lua.go` 不是单一脚本，而是一组业务高频场景脚本封装：

- 排行榜区间查询、成员排名查询
- 排行榜附带 Hash 扩展信息查询
- 分布式锁 / 解锁
- 滑动窗口限流
- 有上限的计数器递增
- 限长队列 push
- 带版本号的 set
- 库存扣减
- HyperLogLog 计数
- 延迟队列弹出
- Bloom Filter add / exists
- Redis 自增 ID
- 代扣幂等号计数

### 8.4 几个代表性用法

排行榜：

```go
info, err := wd.InsRedis.LuaRedisZSetGetMemberScoreAndRankDesc("rank:score", "u_1001")
list, err := wd.InsRedis.LuaRedisZSetGetTargetKeyAndStartToEndRankByScoreDesc("rank:score", 0, 9, "u_1001")
_ = info
_ = list
_ = err
```

限流：

```go
count, err := wd.InsRedis.LuaRedisRateLimit("api:user:1001", 60, 20)
if err != nil {
    return err
}
if count > 20 {
    return wd.MsgErrBadRequest("请求过于频繁")
}
```

库存扣减：

```go
result, err := wd.InsRedis.LuaRedisDecrStock("stock:sku:1", 1)
if err != nil {
    return err
}
if !result.Success {
    return wd.MsgErrBadRequest("库存不足")
}
```

---

## 9. 定时任务、Casbin 权限与 Elasticsearch

### 9.1 定时任务 `cron_task.go`

初始化：

```go
err := wd.InitCronJob(
    wd.WithBeforeJobRuns(func(jobID uuid.UUID, jobName string) {
        println("before", jobName)
    }),
)
if err != nil {
    panic(err)
}
```

常用方法：

- `RunJobEveryDuration(...)`
- `RunJobAtTime(...)`
- `RunJobEveryDay(...)`
- `RunJobCrontab(...)`
- `RunJobEveryDurationTheOne(...)` / `RunJobCrontabTheOne(...)`

带 `TheOne` 后缀的方法会依赖 Redis 锁，避免多实例重复注册任务。

### 9.2 Casbin 权限 `casbin.go`

初始化：

```go
if err := wd.InitCasbin(); err != nil {
    panic(err)
}
if err := wd.InsCasbin.InitCasbinRule(); err != nil {
    panic(err)
}
```

Gin 鉴权中间件：

```go
r.Use(wd.InsCasbin.CustomGinMiddleware(func(c *gin.Context) (string, error) {
    return "admin", nil
}))
```

常用方法：

- `CustomAddPoliciesEx`
- `CustomRemovePoliciesEx`
- `CustomAddRolesForUser`
- `CustomGetPermissionsForRole`
- `CustomGetUserAllInfo`

### 9.3 Elasticsearch `es.go`

```go
err := wd.InitEs(
    wd.WithEsConfigAddresses("http://127.0.0.1:9200"),
    wd.WithEsConfigBatchIndex("audit-log"),
)
if err != nil {
    panic(err)
}

_ = wd.InsEs.CustomBulkInsertData(map[string]any{"id": 1, "msg": "hello"})
_ = wd.InsEs.CustomBulkClose()
```

适合日志归档、搜索索引、批量同步等场景。

---

## 10. 短信服务 `sms.go`

当前仓库里保留的业务能力只剩短信服务，对外统一通过根目录下的 `sms.go` 提供：

- `wd.NewSMS`
- `wd.NewSMSWithAccessKey`
- `wd.NewSMSWithClient`

### 10.1 初始化

通过凭证配置初始化：

```go
svc, err := wd.NewSMS(wd.SMSConfig{
    CredentialConfig: new(credential.Config).
        SetType("access_key").
        SetAccessKeyId("access-key-id").
        SetAccessKeySecret("access-key-secret"),
})
```

直接通过 AccessKey 初始化：

```go
svc, err := wd.NewSMSWithAccessKey("access-key-id", "access-key-secret")
```

复用现有客户端：

```go
svc, err := wd.NewSMSWithClient(dysmsClient)
```

### 10.2 发送短信

发送单条短信：

```go
err := svc.SendSimpleMsg(
    "13800138000",
    "测试签名",
    "SMS_123456789",
    `{"code":"9527"}`,
)
```

发送批量短信：

```go
err := svc.SendSimpleBatchMsg(
    `["13800138000","13900139000"]`,
    `["测试签名","测试签名"]`,
    "SMS_123456789",
    `[{"code":"9527"},{"code":"9528"}]`,
)
```

## 11. Excel 导入导出工具


### 13.1 导出 `excel_export.go`

导出依赖 `excel` tag，格式支持：

- `excel:"order_no"`
- `excel:"order_no,title:订单号"`

示例：

```go
type OrderRow struct {
    OrderNo   string    `excel:"order_no,title:订单号"`
    Username  string    `excel:"username,title:用户名"`
    Amount    float64   `excel:"amount,title:金额"`
    CreatedAt time.Time `excel:"created_at,title:创建时间"`
}

exporter := wd.InitExcelExporter(
    wd.WithExcelExporterSheetName("订单列表"),
    wd.WithExcelExporterColumnWidths(map[string]float64{
        "order_no":   24,
        "username":   18,
        "created_at": 22,
    }),
)

err := exporter.ExportToFile(rows, "./orders.xlsx")
```

常用方法：

- `ExportToFile`
- `ExportToBuffer`
- `ExportToExcelizeFile`
- `ExportToSheet`
- `GetStats`

### 13.2 导入 `excel_mapper.go`

同样依赖 `excel` tag，把 Excel 行映射到结构体切片。

```go
type UserRow struct {
    Name     string     `excel:"姓名"`
    Age      int        `excel:"年龄"`
    Mobile   *string    `excel:"手机号"`
    JoinedAt *time.Time `excel:"入职时间"`
}

mapper := wd.InitExcelMapper(
    wd.WithExcelMapperHeaderRow(1),
    wd.WithExcelMapperDataStartRow(2),
    wd.WithExcelMapperStrictMode(false),
)

var users []UserRow
if err := mapper.MapToStructs("./users.xlsx", &users); err != nil {
    panic(err)
}

for _, item := range mapper.GetErrors() {
    fmt.Println(item.Error())
}
```

### 13.3 Excel 坐标工具 `excel_math.go`

- `ExcelGetPosition(row, col)`
- `ExcelGetPositionBatch(...)`
- `ExcelColumnToIndex("AA")`
- `ExcelParsePosition("B12")`
- `ExcelParsePositionUnsafe("B12")`

适合自己写 Excelize 逻辑时做单元格坐标计算。

---

## 14. 文件、HTTP、加密、随机串等常用工具

### 14.1 配置与文件上传 `file.go`

- `InitConfig(path, &cfg)`：读取配置文件到结构体
- `ReadFileContent(path)`：读取文件内容
- `GetFileContentType(bytes)`：探测 MIME
- `GetFileNameType(fileName)`：拿扩展名
- `UploadFileToTargetURL(...)`：把 `multipart.FileHeader` 转发上传到目标服务

上传示例：

```go
var resp struct {
    URL string `json:"url"`
}

err := wd.UploadFileToTargetURL(
    wd.WithUploadFileValue(fileHeader),
    wd.WithUploadFileURL("https://upload.example.com/file"),
    wd.WithUploadFileToken(token),
    wd.WithUploadFileResp(&resp),
)
```

### 14.2 HTTP 请求工具 `resty.go`

- `RestyClient()`：默认单例客户端
- `R()`：创建请求
- `RGet(...)`
- `RPost(...)`

示例：

```go
var out struct {
    Code int `json:"code"`
}
err := wd.RPost(
    map[string]string{"Content-Type": "application/json"},
    map[string]any{"name": "demo"},
    "https://api.example.com/create",
    &out,
)
```

### 14.3 加密与密码工具 `encrypt.go`

- `EncryptData(data, customKeyFunc)`：序列化后做 AES-GCM 加密
- `PasswordEncryption(password)`：bcrypt 哈希
- `PasswordCompare(password, hashed)`：密码校验
- `PasswordValidateStrength(password, minLen, maxLen)`：强度校验

### 14.4 随机与 ID 工具 `random.go` / `snowflake.go`

- `GetUUID()`
- `GetXID()`
- `InitSnowflakeWorker(workerID)`
- `GetSnowflakeID()`
- `RandomString(length)`
- `RandomStringWithPrefix(...)`
- `RandomIntRange(left, right)`
- `RandomExcludeErrorPronCharacters(...)`

### 14.5 金额工具 `decimal.go`

- `DecimalYuanToFen`
- `DecimalFenToYuan`
- `DecimalFenToYuanStr`
- `DecimalAddsSubsGteZero`

### 14.6 字符串与脱敏 `string.go`

- `ValidateChineseMobile`
- `ValidateChineseIDCard`
- `MaskMobile`
- `MaskIDCard`
- `MaskUsername`
- `GetGenderFromIDCard`
- `ReplacePathParamsFast`

### 14.7 模板与差异比较

`template.go`：

- `TemplateReplace(templateText, data)`

`obj_diff.go`：

- `DiffReturnLogs`
- `DiffReturnSemanticLogs`
- `DiffText`
- `DiffReturnHtml`
- `DiffReturnColorText`

适合“更新前后差异审计”“操作日志语义化输出”之类的场景。

### 14.8 JSON / 集合 / Context 小工具

`gjson.go`：

- `JsonGetValue(jsonStr, key)`

`lo.go`：

- `LoMap`
- `LoSliceToMap`
- `LoTernary`
- `LoWithout`
- `LoUniq`
- `LoToPtr`
- `LoFromPtr`

`context.go`：

- `Context(ttl...)`
- `DurationSecond(second)`

---


## 附录：按文件查 API

如果你已经知道自己要找哪个文件，这里可以直接反查主要导出入口。这个附录不追求把每个导出符号全部列完，而是优先列“真正会被业务项目直接调用”的 API。

### 服务主干

| 文件 | 主要 API |
| --- | --- |
| `gin_engine.go` | `PublicRoutes.Append`、`PrivateRoutes.Append`、`WithGinRouterPrefix`、`WithGinRouterAuthHandler`、`WithGinRouterGlobalMiddleware`、`WithGinRouterModel` |
| `http_server.go` | `InitHTTPServerAndStart`、`NewHTTPServer`、`(*HTTPServer).Start`、`StartAsync`、`Wait` |
| `middleware_log.go` | `MiddlewareLogger`、`BeginStageTiming`、`WriteGinInfoLog`、`WriteGinWarnLog`、`WriteGinErrAnyLog`、`GinLogSetModuleName`、`GinLogSetOptionName` |
| `middleware_trace_id.go` | `MiddlewareTraceID`、`GetTraceID` |
| `middleware_request_time.go` | `MiddlewareRequestTime` |
| `middleware_recovery.go` | `MiddlewareRecovery` |
| `middleware_cors.go` | `Cors` |

### 认证与响应

| 文件 | 主要 API |
| --- | --- |
| `auth_jwt.go` | `NewGinJWTMiddleware`、`(*GinJWTMiddleware).MiddlewareFunc`、`LoginHandler`、`RefreshHandler`、`TokenGenerator`、`ParseTokenString`、`ExtractClaimsAs`、`GetIdentityAs`、`GetToken` |
| `auth_jwt_options.go` | `WithJWTRealm`、`WithJWTKey`、`WithJWTTimeout`、`WithJWTMaxRefresh`、`WithJWTIdentityKey`、`WithJWTTokenLookup`、`WithJWTCookie`、`WithJWTRSA` |
| `response.go` | `ResponseSuccess`、`ResponseSuccessMsg`、`ResponseSuccessToken`、`ResponseSuccessEncryptData`、`ResponseError`、`ResponseParamError`、`ConvertToAppError`、各类 `MsgErr*` |
| `params_verify.go` | `TranslateError`、`CreateRequiredError`、`CreateTypeError` |
| `gin_param.go` | `GinQueryDefault`、`GinQueryRequired`、`GinPathRequired` |

### PATCH、查询参数与文件表单

| 文件 | 主要 API |
| --- | --- |
| `patch_field.go` | `Field[T]`、`(Field[T]).IsSet`、`HasValue` |
| `patch_field_assign.go` | `PatchUpdateSimple`、`PatchUpdate` |
| `params_precompiled.go` | `ReqRange[T]`、`ReqKeyword`、`ReqPageSize`、`ReqFile`、`ReqFiles`、`ApplyPage`、`FilesUploadGoroutine` |
| `binding_patch_validator.go` | Gin `binding` 与 `Field[T]` 协作支持（通常无需手动调用） |

### 数据库、缓存、搜索与调度

| 文件 | 主要 API |
| --- | --- |
| `gorm.go` | `InitGormDB`、`GormDefaultLogger`、`WrapGormLoggerWithRequestLogger`、`WithGormConfig*` |
| `gen.go` | `(*GormClient).Gen`、`WithGenOutFilePath`、`WithGenUseTablesName`、`WithGenTableColumnType`、`WithGenGlobalColumnTypeAddDatatypes` |
| `gen_field.go` | `GenJSONArrayQuery`、`GenJSONArrayQueryContainsValue`、`GenCustomTimeBetween`、`GenNewBetween` |
| `redis.go` | `InitRedis`、`(*RedisConfig).NewLock`、`SetCaptcha`、`GetCaptcha`、`DelCaptcha`、`FindAllBitMapByTargetValue`、`WithRedis*` |
| `redis_lua.go` | `LuaRedisRateLimit`、`LuaRedisDecrStock`、`LuaRedisIncrWithLimit`、`LuaRedisLeaderboardIncr`、`LuaRedisDistributedLock`、`LuaRedisBloomAdd`、`LuaRedisID` |
| `cron_task.go` | `InitCronJob`、`RunJobEveryDuration`、`RunJobCrontab`、`RunJobEveryDurationTheOne`、`Start`、`Stop` |
| `casbin.go` | `InitCasbin`、`(*CachedEnforcer).InitCasbinRule`、`CustomGinMiddleware`、`CustomAddPoliciesEx`、`CustomAddRolesForUser` |
| `es.go` | `InitEs`、`CustomBulkInsertData`、`CustomBulkClose`、`CustomBulkStats` |

### 时间、Excel 与通用工具

| 文件 | 主要 API |
| --- | --- |
| `sql_type.go` | `DateTime`、`DateOnly`、`TimeOnly`、`TimeHM` |
| `time.go` | `Now`、`NowAsDateTime`、`ParseDateTimeValue`、`NewDateOnlyString`、`TodayRange`、`CurrentMonthRange`、`HasTimeConflict` |
| `excel_export.go` | `InitExcelExporter`、`ExportToFile`、`ExportToBuffer`、`ExportToExcelizeFile`、`GetStats` |
| `excel_mapper.go` | `InitExcelMapper`、`MapToStructs`、`GetErrors`、`ClearErrors` |
| `excel_math.go` | `ExcelGetPosition`、`ExcelGetPositionBatch`、`ExcelColumnToIndex`、`ExcelParsePosition` |
| `file.go` | `InitConfig`、`ReadFileContent`、`GetFileContentType`、`GetFileNameType`、`UploadFileToTargetURL` |
| `resty.go` | `RestyClient`、`R`、`RPost`、`RGet` |
| `encrypt.go` | `EncryptData`、`PasswordEncryption`、`PasswordCompare`、`PasswordValidateStrength` |
| `random.go` | `GetUUID`、`GetXID`、`InitSnowflakeWorker`、`GetSnowflakeID`、`RandomString`、`RandomIntRange` |
| `decimal.go` | `DecimalYuanToFen`、`DecimalFenToYuan`、`DecimalFenToYuanStr` |
| `string.go` | `ValidateChineseMobile`、`ValidateChineseIDCard`、`MaskMobile`、`MaskIDCard`、`MaskUsername` |
| `obj_diff.go` | `DiffReturnLogs`、`DiffReturnSemanticLogs`、`DiffText`、`DiffReturnHtml` |
| `template.go` | `TemplateReplace` |
| `lo.go` | `LoMap`、`LoSliceToMap`、`LoTernary`、`LoUniq`、`LoToPtr`、`LoFromPtr` |
| `gjson.go` | `JsonGetValue` |
| `context.go` | `Context`、`DurationSecond` |
| `signal.go` | `InsGlobalHook`、`(*SignalHook).AppendFun`、`Trigger`、`Wait` |

### 业务能力入口

| 文件/包 | 主要 API |
| --- | --- |
| `sms.go` | `NewSMS`、`NewSMSWithAccessKey`、`NewSMSWithClient`、`SendMsg`、`SendSimpleMsg`、`SendBatchSms` |


## 15. 其他基础工具索引

下面这些工具不一定需要单独展开成章节，但在实际项目里都很实用：

| 文件 | 主要能力 |
| --- | --- |
| `model.go` | 常见 GORM 时间字段结构体片段，如 `StructGormIDDateTime` |
| `custom_const.go` | 公共常量、上下文 key、tag key |
| `binding_patch_validator.go` | 让 `Field[T]` 与 Gin `binding` 正常协作 |
| `other_errors.go` | GORM 常见错误判断，如记录不存在、唯一键冲突 |
| `signal.go` | 全局优雅关闭钩子 `InsGlobalHook` |
| `middleware_cors.go` | CORS 中间件 `Cors(...)` |
| `response.go` | `ReturnAppErr` 等错误转换辅助 |
| `gen_field.go` | `gorm/gen` 条件表达式辅助 |
| `internal/xclients/*` | 微信/支付宝 SDK 初始化内部封装 |
| `internal/xhelper/nil.go` | 处理 interface nil 判断 |

---

## 16. 推荐阅读顺序

如果你第一次接触这个库，建议这样看：

1. 先看“快速接入”了解整体风格；
2. 再看 `HTTP + JWT + 响应 + GORM`，这是最基础的主干；
3. 如果你需要短信能力，再看 `sms.go` 与 `docs/message.md`；
4. 如果你做后台业务系统，再看 `PATCH`、`ReqRange`、`Excel`、`Redis Lua`；
5. 最后按需查“通用工具索引”。

## 17. 仓库内示例与测试

仓库里已经有一些非常值得参考的示例：

- `test/projectflow/message_flow_test.go`：短信模块装配方式
- `test/httpt/h_test.go`：`ReqRange`、`Field[T]`、`PatchUpdate`、`gorm/gen` 的组合用法

如果你不知道某个工具在真实项目中应该如何拼装，优先看这些测试文件。

## 18. 总结

如果把 `wd` 当成一个工具箱来理解，可以把它分成三层：

- 第一层：服务基础设施工具
  - HTTP、Gin、日志、响应、JWT、GORM、Redis、Cron、Casbin、ES
- 第二层：业务通用模型工具
  - PATCH、分页、范围查询、时间类型、Excel、文件上传
- 第三层：业务场景工具包
  - 阿里云短信发送

所以最合适的使用方式不是“只引一个函数”，而是把它作为项目的统一工具层：

- 服务入口统一走 `InitHTTPServerAndStart`
- 认证统一走 `NewGinJWTMiddleware`
- 响应统一走 `ResponseSuccess` / `ResponseError`
- 数据访问统一走 `InitGormDB` + `InsDB.Gen`
- 缓存与分布式能力统一走 `InitRedis`
- 短信能力统一走 `sms.go`

这样项目的接入风格会更统一，后续维护成本也会低很多。
