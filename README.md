# wd

`wd` 是一个面向 Go 服务的通用组件库，聚合了 HTTP 服务启动、Gin 中间件、JWT 认证、GORM/Redis 封装、Resty 客户端、Excel/加密/随机数等常用能力，帮助快速搭建可观测、可扩展的业务应用。

## 功能亮点
- **一体化 HTTP 启动器**（`http.go` + `gin_engine.go`）提供公共/私有路由注册、全局/认证中间件注入、健康检查、优雅关闭及运行参数（超时、前缀、日志等）的统一配置接口。
- **JWT 认证链路**（`auth_jwt.go`）具备登录/退出/刷新处理器、灵活的 Token 提取策略、Cookie 同步、可插拔 payload/授权钩子，并可自动接入 HTTP 启动器的私有路由。
- **数据访问与缓存**：`gorm.go` 暴露 `InitGormDB`、`GormDefaultLogger`，`redis.go` 封装 `redis.UniversalClient`、分布式锁（redsync）及多种配置项，便于以一致方式管理数据库和 Redis。
- **可观测中间件**：`middleware_trace_id.go`、`middleware_request_time.go`、`middleware_log.go` 等提供链路日志、TraceID、请求耗时、异常恢复等通用能力；其中请求日志默认只保留请求摘要和业务主动写入的日志条目。
- **网络与工具集**：`resty.go` 提供默认 HTTP 客户端，`response.go`/`request.go` 统一请求/响应结构，`password.go`、`encrypt.go`、`random.go`、`snowflake.go` 等实现密码、加解密、随机 ID、ID 生成器。
- **场景扩展**：`auth/`、`payment/`、`message/`、`excel_*` 等子目录按能力组织登录、支付、消息等能力，并在各能力下继续按渠道拆分，便于按需接入。

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
            payload, _ := wd.ExtractClaimsAs[accountPayload](c)
            c.JSON(http.StatusOK, gin.H{
                "payload": payload,
                "token":   wd.GetToken(c),
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

## 能力模块详解
新的扩展能力按 `auth/`、`payment/`、`message/` 三个模块组织，每个模块下再按渠道拆分。对使用者来说，推荐的心智模型是：

1. 先按业务能力选择模块：登录、支付、消息；
2. 再按接入渠道选择具体子包：微信、支付宝、公众号、企业微信、短信；
3. 最后决定是让 `wd` 自己初始化第三方 SDK，还是复用你项目里已有的 SDK 客户端。

三个模块都遵循同样的接入风格：

- `New(...)`：按配置初始化第三方客户端和能力服务
- `NewWithClient(...)`：复用调用方已经初始化好的第三方客户端
- `RegisterRoutes(...)`：把该能力的 HTTP 路由挂到 gin 路由组
- `Register(...)`：按能力批量注册多个服务

你可以直接参考 `test/projectflow/` 下的示例：

- `test/projectflow/auth_flow_test.go`
- `test/projectflow/payment_flow_test.go`
- `test/projectflow/message_flow_test.go`
- `test/projectflow/wechatmini_real_flow_test.go`

最后一份 `wechatmini_real_flow_test.go` 演示了更接近真实商城项目的流程：微信小程序登录、查人、注册、签发 JWT、创建订单、发起支付、支付回调改订单状态。

### 认证模块 `auth/`
`auth/` 负责第三方登录场景下的“身份换业务用户、业务用户换 token”这条链路。

当前已经内置两个渠道：

- `auth/wechatmini`
- `auth/alipaymini`

#### 目录说明

```text
auth/
  register.go        # 统一注册入口
  types.go           # 通用身份类型与 UserHandler 接口
  response.go        # 统一 token / 响应输出辅助
  wechatmini/        # 微信小程序登录
  alipaymini/        # 支付宝小程序登录
```

#### 通用接口
认证模块最核心的是 `auth.UserHandler`。第三方渠道负责把 `UnionID/OpenID/手机号` 等外部身份信息整理出来，业务方只需要实现“查用户、建用户、发 token”这三个步骤：

```go
type UserHandler interface {
    FindUser(ctx context.Context, identity auth.Identity) (user any, exists bool, err error)
    CreateUser(ctx context.Context, identity auth.Identity) (user any, err error)
    GenerateToken(ctx context.Context, user any, identity auth.Identity, sessionValue string) (data any, err error)
}
```

其中 `auth.Identity` 会统一承接：

- `Provider`
- `UnionID`
- `OpenID`
- `PhoneNumber`
- `ClientIP`

也就是说，`wd` 负责把第三方身份“解出来”，你负责把它“落到业务库里”。

#### 微信小程序登录用法
下面示例更接近真实项目：用户通过微信 `code` 登录，系统根据微信身份查会员，不存在则自动注册并签发 JWT。

```go
package main

import (
    "context"
    "time"

    "github.com/gin-gonic/gin"
    wd "github.com/loveyu233/wd"
    "github.com/loveyu233/wd/auth"
    authwechatmini "github.com/loveyu233/wd/auth/wechatmini"
)

type member struct {
    ID          int64
    UnionID     string
    OpenID      string
    PhoneNumber string
    Nickname    string
}

type claims struct {
    MemberID int64  `json:"member_id"`
    OpenID   string `json:"open_id"`
    Nickname string `json:"nickname"`
}

type memberAuthHandler struct {
    jwt *wd.GinJWTMiddleware
}

func (h *memberAuthHandler) FindUser(ctx context.Context, identity auth.Identity) (any, bool, error) {
    // 真实项目里建议先按 UnionID 查，不存在时再按 OpenID 查
    return nil, false, nil
}

func (h *memberAuthHandler) CreateUser(ctx context.Context, identity auth.Identity) (any, error) {
    // 真实项目里这里会落库
    return &member{
        ID:          1001,
        UnionID:     identity.UnionID,
        OpenID:      identity.OpenID,
        PhoneNumber: identity.PhoneNumber,
        Nickname:    "微信用户",
    }, nil
}

func (h *memberAuthHandler) GenerateToken(ctx context.Context, user any, identity auth.Identity, sessionValue string) (any, error) {
    memberData := user.(*member)
    token, _, err := h.jwt.TokenGenerator(memberData)
    return token, err
}

func buildAuthModule() (*authwechatmini.Service, error) {
    jwtMW, err := wd.NewGinJWTMiddleware(
        func(c *gin.Context) (*member, error) {
            return nil, wd.MsgErrBadRequest("这个示例使用 TokenGenerator 发 token，不走 LoginHandler")
        },
        func(data *member) claims {
            return claims{
                MemberID: data.ID,
                OpenID:   data.OpenID,
                Nickname: data.Nickname,
            }
        },
        func(c *gin.Context, payload claims) (any, error) {
            return payload.MemberID, nil
        },
        wd.WithJWTKey([]byte("mall-demo-secret")),
        wd.WithJWTTimeout(2*time.Hour),
        wd.WithJWTIdentityKey("member_id"),
    )
    if err != nil {
        return nil, err
    }

    return authwechatmini.New(authwechatmini.Config{
        AppID:  "wx-app-id",
        Secret: "wx-secret",
    }, &memberAuthHandler{jwt: jwtMW})
}

func registerAuth(r *gin.RouterGroup) error {
    svc, err := buildAuthModule()
    if err != nil {
        return err
    }

    auth.Register(r.Group("/auth/wechatmini"), svc)
    return nil
}
```

#### 支付宝小程序登录用法
`auth/alipaymini` 的业务侧接口和微信保持一致，差别主要在初始化配置：

```go
svc, err := authalipaymini.New(authalipaymini.Config{
    AppID:                "alipay-app-id",
    AppPrivateKey:        "your-private-key",
    AESKey:               "your-aes-key",
    AppPublicKeyFilePath: "./cert/alipay/appPublicKey.crt",
    AliPublicKeyFilePath: "./cert/alipay/alipayPublicKey_RSA2.crt",
    AliRootKeyFilePath:   "./cert/alipay/alipayRootCert.crt",
}, handler)
```

如果你的项目已经自己维护了支付宝 SDK 客户端，也可以直接复用：

```go
svc, err := authalipaymini.NewWithClient(alipayClient, handler, true)
```

#### 认证模块推荐实践

- `FindUser` 里优先按 `UnionID` 查，`UnionID` 缺失时再考虑 `OpenID`
- `CreateUser` 里尽量把 `UnionID/OpenID/手机号` 一次性保存完整，避免后续补绑定复杂化
- `GenerateToken` 建议统一复用 `wd.NewGinJWTMiddleware(...).TokenGenerator(...)`
- 如果业务方已经有统一 JWT 中间件，认证模块只负责发 token，不负责自己重复造一套鉴权体系

### 支付模块 `payment/`
`payment/` 负责“组装支付请求、接收回调、驱动订单状态流转”。

当前已经内置两个渠道：

- `payment/wechat`
- `payment/alipay`

#### 目录说明

```text
payment/
  register.go        # 统一注册入口
  wechat/            # 微信支付
  alipay/            # 支付宝支付
```

#### 微信支付接口约定
微信支付核心由 `payment/wechat.Handler` 定义：

```go
type Handler interface {
    BuildPayRequest(c *gin.Context) (*paymentwechat.PayRequest, error)
    BuildRefundRequest(c *gin.Context) (*paymentwechat.RefundRequest, error)
    OnPaymentNotify(ctx context.Context, orderID, attach string) error
    OnRefundNotify(ctx context.Context, refundOrderID string) error
}
```

这意味着：

- `wd` 负责和微信支付 SDK 交互
- 你负责从业务请求里找到订单、校验归属、拼出支付参数
- 回调到来后，你负责按订单号改业务状态

#### 微信支付接入示例
下面是一个更贴近商城项目的接入方式：

```go
package main

import (
    "context"

    "github.com/gin-gonic/gin"
    wd "github.com/loveyu233/wd"
    "github.com/loveyu233/wd/payment"
    paymentwechat "github.com/loveyu233/wd/payment/wechat"
)

type order struct {
    OrderNo   string
    MemberID  int64
    Title     string
    AmountFen int64
    Status    string
}

type wechatPayHandler struct {
    orders map[string]*order
}

func (h *wechatPayHandler) BuildPayRequest(c *gin.Context) (*paymentwechat.PayRequest, error) {
    var req struct {
        OrderNo string `json:"order_no" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        return nil, err
    }

    claims, err := wd.ExtractClaimsAs[struct {
        MemberID int64 `json:"member_id"`
        OpenID   string `json:"open_id"`
    }](c)
    if err != nil {
        return nil, err
    }

    orderData := h.orders[req.OrderNo]
    if orderData == nil {
        return nil, wd.MsgErrNotFound("订单不存在")
    }
    if orderData.MemberID != claims.MemberID {
        return nil, wd.MsgErrForbiddenAuth("订单不属于当前用户")
    }
    if orderData.Status != "WAIT_PAY" {
        return nil, wd.MsgErrBadRequest("订单当前状态不可支付")
    }

    return &paymentwechat.PayRequest{
        Price:       orderData.AmountFen,
        Description: orderData.Title,
        OpenID:      claims.OpenID,
        Attach:      `{"scene":"mall_checkout","order_no":"` + orderData.OrderNo + `"}`,
        OutTradeNo:  orderData.OrderNo,
    }, nil
}

func (h *wechatPayHandler) BuildRefundRequest(c *gin.Context) (*paymentwechat.RefundRequest, error) {
    return nil, wd.MsgErrBadRequest("示例省略退款")
}

func (h *wechatPayHandler) OnPaymentNotify(ctx context.Context, orderID, attach string) error {
    orderData := h.orders[orderID]
    if orderData == nil {
        return wd.MsgErrNotFound("订单不存在")
    }
    orderData.Status = "PAID"
    return nil
}

func (h *wechatPayHandler) OnRefundNotify(ctx context.Context, refundOrderID string) error {
    return nil
}

func registerPayment(r *gin.RouterGroup) error {
    svc, err := paymentwechat.New(paymentwechat.Config{
        AppID:       "wx-app-id",
        MchID:       "mch-id",
        MchApiV3Key: "mch-api-v3-key",
        NotifyURL:   "https://api.example.com/api/payment/wechat/notify/payment",
    }, &wechatPayHandler{
        orders: map[string]*order{},
    })
    if err != nil {
        return err
    }

    payment.Register(r.Group("/payment/wechat"), svc)
    return nil
}
```

#### 支付宝支付接入方式
`payment/alipay` 和微信支付的模式一致，区别主要在配置和通知载荷类型：

```go
svc, err := paymentalipay.New(paymentalipay.Config{
    AppID:                "alipay-app-id",
    AppPrivateKey:        "your-private-key",
    AESKey:               "your-aes-key",
    AppPublicKeyFilePath: "./cert/alipay/appPublicKey.crt",
    AliPublicKeyFilePath: "./cert/alipay/alipayPublicKey_RSA2.crt",
    AliRootKeyFilePath:   "./cert/alipay/alipayRootCert.crt",
    NotifyURL:            "https://api.example.com/api/payment/alipay/notify",
}, handler)
```

支付宝回调里，业务方会拿到的是结构化的 `paymentalipay.PaymentNotify` / `paymentalipay.RefundNotify`，不用再自己做签名验签和表单字段拆解。

#### 支付模块推荐实践

- `BuildPayRequest` 里务必校验订单归属、订单状态、金额来源，不要相信前端传来的金额
- `OnPaymentNotify` / `OnRefundNotify` 要做幂等处理，避免第三方重复通知导致重复改状态
- 回调成功和失败一定要围绕“订单状态是否真正落库成功”决定，而不是只看参数是否解析成功
- 如果业务方已经自己初始化了第三方支付客户端，可以优先用 `NewWithClient(...)`

### 消息模块 `message/`
`message/` 负责主动消息、回调消息、订阅消息和短信通知。

当前已经内置四个渠道：

- `message/officialaccount`
- `message/miniprogram`
- `message/qywx`
- `message/sms`

#### 目录说明

```text
message/
  register.go              # 统一注册入口
  officialaccount/         # 微信公众号消息
  miniprogram/             # 微信小程序订阅消息
  qywx/                    # 企业微信机器人消息
  sms/                     # 阿里云短信
```

#### 公众号消息用法
公众号消息既有被动回调，也有主动推送。业务方实现 `message/officialaccount.Handler` 即可：

```go
type Handler interface {
    OnSubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
    OnUnsubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
    BuildPushRequest(c *gin.Context) (toUsers []string, message string, err error)
}
```

初始化和注册：

```go
svc, err := officialaccount.New(officialaccount.Config{
    AppID:         "wx-official-app-id",
    AppSecret:     "wx-official-secret",
    MessageToken:  "server-token",
    MessageAESKey: "server-aes-key",
}, handler)
if err != nil {
    return err
}

message.Register(r.Group("/message/officialaccount"), svc)
```

这样会注册三类路由：

- `GET /callback`：公众号回调验证
- `POST /callback`：公众号消息/事件通知
- `POST /push`：主动推送消息

#### 小程序订阅消息用法
`message/miniprogram` 不强依赖 gin 路由，更适合在下单成功、发货成功、退款成功这些业务节点里直接调用：

```go
svc, err := miniprogram.New(miniprogram.Config{
    AppID:  "wx-app-id",
    Secret: "wx-secret",
})
if err != nil {
    return err
}

_, err = svc.SubscribeMessageSend(ctx, miniprogram.SubscribeContent{
    ToUserOpenID: "user-open-id",
    TemplateID:   "template-id",
    Page:         "pages/order/detail?id=1001",
    State:        miniprogram.StateFormal,
    Data: map[string]map[string]any{
        "thing1": {"value": "订单支付成功"},
        "amount2": {"value": "199.00"},
    },
})
```

#### 企业微信机器人用法
企业微信机器人适合发布部署通知、风控告警、运营提醒等场景：

```go
svc, err := qywx.New(qywx.Config{
    WebhookKey: "robot-webhook-key",
})
if err != nil {
    return err
}

_, err = svc.SendText("发布成功")
_, err = svc.SendMarkdown("**库存告警**：商品 A 库存不足")
```

也可以先上传文件，再发送文件消息：

```go
_, err = svc.SendFile("/tmp/report.xlsx", qywx.MediaTypeFile)
```

#### 短信用法
短信能力更适合验证码、营销通知、支付成功通知等场景。支持按凭证初始化，也支持复用现有客户端：

```go
svc, err := sms.NewWithAccessKey("access-key-id", "access-key-secret")
if err != nil {
    return err
}

err = svc.SendSimpleMsg(
    "13800138000",
    "短信签名",
    "SMS_123456789",
    `{"code":"9527"}`,
)
```

#### 消息模块推荐实践

- 回调型能力（公众号）建议始终通过路由注册方式接入，便于统一日志和中间件
- 主动消息能力（小程序订阅消息、企业微信机器人、短信）建议在业务层直接注入 service 使用
- 对外消息通常是副作用操作，推荐在业务层做好失败重试、熔断和日志记录
- 如果项目里已经统一管理第三方 SDK 客户端，优先使用各模块的 `NewWithClient(...)`

### 阶段耗时记录
如果你想在单个请求内记录某一段业务操作的耗时，可以使用 `BeginStageTiming`：

```go
rg.POST("/user/:id", func(c *gin.Context) {
    stage := wd.BeginStageTiming(c, "修改数据库")

    // 执行数据库更新
    if err := updateUser(c); err != nil {
        wd.ResponseError(c, err)
        return
    }

    stage.Commit()
    wd.ResponseSuccessMsg(c, "ok")
})
```

调用 `Commit()` 后，请求日志中会追加类似 `阶段[修改数据库]耗时=12.34ms` 的记录。

### 请求日志说明
当前 `MiddlewareLogger` 的行为已经简化，请求日志默认只包含以下基础信息：

- `method`
- `url`
- `ip`
- `module`
- `option`

如果你希望在单个请求内追加业务日志，请在处理流程中主动调用：

- `WriteGinInfoLog`
- `WriteGinDebugLog`
- `WriteGinWarnLog`
- `WriteGinErrLog`
- `WriteGinInfoAnyLog`
- `WriteGinDebugAnyLog`
- `WriteGinWarnAnyLog`
- `WriteGinErrAnyLog`

其中 `WriteGin*AnyLog` 适合直接记录结构体、切片、map 等任意对象，例如把本次请求的响应结构体直接写入请求日志。

如果你配置了 `WithGinRouterLogSaveLog`，持久化回调收到的 `ReqLog` 也只会保留上述基础请求摘要，以及业务主动写入的 `Logs`。中间件不再主动解析请求体、响应体，也不再区分 GET 请求或精简日志模式。



 ## 目录速览
      .
      ├── auth_jwt.go            # JWT 中间件及登录/刷新/退出处理器
      ├── auth_jwt_options.go     # JWT 中间件函数选项（WithJWT* 系列）
      ├── gin_engine.go          # Gin Router 构建、公共/私有路由注册与中间件装配
      ├── http.go                # HTTP Server 启动器、优雅关闭、默认认证中间件挂载
      ├── middleware_trace_id.go # TraceID 注入
      ├── middleware_request_time.go # 请求耗时统计
      ├── middleware_log.go      # 链路日志与请求摘要记录
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
      ├── auth/                  # 认证能力，按渠道拆分为 wechatmini、alipaymini
      ├── payment/               # 支付能力，按渠道拆分为 wechat、alipay
      ├── message/               # 消息能力，按渠道拆分为 officialaccount、miniprogram、qywx、sms
      ├── internal/xclients/     # 第三方客户端共享初始化辅助
      ├── go.mod / go.sum        # 依赖与版本
      └── README.md              # 项目说明（本次已更新）

## 贡献

欢迎通过 Issue/PR 提交新功能或修复，提交前请确保：
1. 新增/修改代码已通过 `go test ./...`；
2. 如引入新特性，请在 README 或相应示例中补充说明；
3. 遵循现有代码风格（使用 `gofmt`）。
