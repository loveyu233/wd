# 认证模块

`wd/auth` 解决的不是“通用 JWT 登录”问题，而是第三方身份接入场景中的这一整段链路：

1. 前端把三方登录凭证传给后端；
2. 后端向第三方平台换取身份信息；
3. 业务系统根据三方身份查找或创建自己的用户；
4. 业务系统签发自己的 token / session 响应。

当前内置两个渠道：

- `auth/wechatmini`
- `auth/alipaymini`

如果你做的是小程序商城、会员系统、内容平台，通常会直接用到这个模块。

## 一、核心设计

### 1. 统一身份模型 `auth.Identity`

第三方登录模块先把渠道侧身份收敛成统一结构：

```go
type Identity struct {
    Provider    string
    UnionID     string
    OpenID      string
    PhoneNumber string
    ClientIP    string
}
```

这意味着业务层不需要感知微信或支付宝 SDK 的细节，只需要处理：

- `Provider`：登录来源
- `UnionID` / `OpenID`：第三方主键
- `PhoneNumber`：若授权拿到了手机号
- `ClientIP`：来源 IP

### 2. 业务方只实现 `auth.UserHandler`

```go
type UserHandler interface {
    FindUser(ctx context.Context, identity auth.Identity) (user any, exists bool, err error)
    CreateUser(ctx context.Context, identity auth.Identity) (user any, err error)
    GenerateToken(ctx context.Context, user any, identity auth.Identity, sessionValue string) (data any, err error)
}
```

推荐把这三个方法理解成三个职责边界：

- `FindUser`：查“这个三方身份对应哪个业务用户”
- `CreateUser`：查不到时如何注册用户
- `GenerateToken`：注册/登录成功后如何发你的业务 token

也就是说：

- 渠道身份解析由 `wd` 负责
- 你的会员系统、账户系统、注册策略由你负责
- token 内容与签发方式由你负责

## 二、统一接入流程

无论微信还是支付宝，推荐都按下面流程接：

1. 项目启动时初始化对应 `Service`
2. 在路由层调用 `auth.Register(...)`
3. 在 `FindUser` 里按 `UnionID -> OpenID` 顺序查用户
4. 在 `CreateUser` 里落库并建立第三方身份绑定
5. 在 `GenerateToken` 里统一复用 `wd.NewGinJWTMiddleware(...).TokenGenerator(...)`

这样第三方登录模块和你的业务会员体系边界最清晰。

## 三、微信小程序登录 `auth/wechatmini`

### 1. 初始化

```go
svc, err := authwechatmini.New(
    authwechatmini.Config{
        AppID:          "wx-demo",
        Secret:         "secret",
        SaveHandlerLog: true,
    },
    yourUserHandler,
)
```

如果项目里已经统一初始化了小程序 SDK，则直接复用客户端：

```go
svc, err := authwechatmini.NewWithClient(miniClient, yourUserHandler, true)
```

### 2. 路由注册

```go
auth.Register(r.Group("/wechatmini"), svc)
```

默认会注册：

- `POST /wechatmini/login`

### 3. 请求体

```go
type LoginRequest struct {
    Code          string `json:"code" binding:"required"`
    EncryptedData string `json:"encrypted_data"`
    IvStr         string `json:"iv_str"`
}
```

行为分两段：

- 只传 `code`：服务端会先换取 `OpenID/UnionID`，如果业务侧 `FindUser` 查不到用户，会先返回 `open_id`，提示前端继续手机号授权流程
- 同时传 `encrypted_data + iv_str`：服务端会尝试解密手机号，然后调用 `CreateUser`

### 4. Service 额外能力

除了登录路由，`wechatmini.Service` 还封了小程序码能力：

- `CreateQRCode(ctx, pagePath, width)`
- `GetCode(code MiniCode)`
- `GetUnlimitedCode(code *MiniUnlimitedCode)`

适合在商城分享、商品详情、邀请海报等场景直接复用。

### 5. 小程序码示例

```go
code := authwechatmini.NewMiniUnlimitedCode(ctx).
    SetPagePath("pages/goods/detail").
    SetScene("id=1001").
    SetWidth(430).
    SetEnvVersion("release")

resp, err := svc.GetUnlimitedCode(code)
```

## 四、支付宝小程序登录 `auth/alipaymini`

### 1. 初始化

```go
svc, err := authalipaymini.New(
    authalipaymini.Config{
        AppID:                "2021xxxx",
        AppPrivateKey:        "private-key",
        AESKey:               "aes-key-base64",
        AppPublicKeyFilePath: "./appPublicKey.pem",
        AliPublicKeyFilePath: "./alipayPublicKey_RSA2.pem",
        AliRootKeyFilePath:   "./alipayRootCert.crt",
        SaveHandlerLog:       true,
    },
    yourUserHandler,
)
```

若已有支付宝客户端：

```go
svc, err := authalipaymini.NewWithClient(alipayClient, yourUserHandler, true)
```

### 2. 路由注册

```go
auth.Register(r.Group("/alipaymini"), svc)
```

默认会注册：

- `POST /alipaymini/login`

### 3. 请求体

```go
type LoginRequest struct {
    Code          string `json:"code" binding:"required"`
    EncryptedData string `json:"encrypted_data"`
}
```

模块内部会：

- 先通过 `code` 换取 `OpenId/UnionId`
- 若需要注册且前端传了 `EncryptedData`，会尝试解密手机号
- 最终调用你的 `FindUser / CreateUser / GenerateToken`

## 五、推荐的 JWT 结合方式

认证子包最推荐的 token 签发方式，不是自己手写 JWT，而是统一复用 `wd.NewGinJWTMiddleware(...).TokenGenerator(...)`：

```go
type claims struct {
    UserID   uint64 `json:"user_id"`
    Nickname string `json:"nickname"`
}

jwtMW, err := wd.NewGinJWTMiddleware(
    func(c *gin.Context) (*member, error) {
        return nil, wd.MsgErrBadRequest("示例中不走 LoginHandler")
    },
    func(data *member) claims {
        return claims{UserID: data.ID, Nickname: data.Nickname}
    },
    func(c *gin.Context, payload claims) (any, error) {
        return payload.UserID, nil
    },
    wd.WithJWTKey([]byte("secret")),
)

token, _, err := jwtMW.TokenGenerator(memberData)
```

这样你就能保证：

- 小程序登录发的 token
- 普通账号密码登录发的 token
- 管理后台 token

在格式、过期时间、Claims 读取方式上保持一致。

## 六、业务侧最佳实践

### 1. 查人顺序建议

对于微信小程序，推荐：

1. 先按 `UnionID` 查
2. 查不到再按 `OpenID` 查
3. 绑定后补齐 `UnionID`

因为 `UnionID` 更稳定，更适合做跨端统一账户。

### 2. `CreateUser` 不要做太多副作用

建议把 `CreateUser` 控制在“最小可注册闭环”内：

- 写用户主表
- 写三方身份绑定表
- 返回用户对象

其他副作用如欢迎券、营销弹窗、埋点等，尽量在异步流程里做。

### 3. 登录态和注册态分开记录日志

- 登录成功：记录 `provider/openid/user_id`
- 自动注册：额外记录 `phone_number/unionid`
- 登录失败：记录失败节点是 `换 session`、`解密手机号`、`发 token` 还是 `业务落库`

## 七、仓库内可参考示例

- `test/projectflow/auth_flow_test.go`
- `test/projectflow/wechatmini_real_flow_test.go`
- `test/projectflow/shared_test.go`

如果你要在真实商城里接小程序登录，优先看 `wechatmini_real_flow_test.go`。
