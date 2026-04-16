# 认证模块

`wd/auth` 解决的不是“通用 JWT 登录”问题，而是第三方身份接入场景中的这一整段链路：

1. 前端把三方登录凭证传给后端；
2. 后端向第三方平台换取身份信息；
3. 业务系统根据三方身份查找或创建自己的用户；
4. 业务系统签发自己的 token / session 响应。

当前内置渠道：

- `auth/wechatmini`

如果你做的是小程序商城、会员系统、内容平台，通常会直接用到这个模块。

## 一、核心设计

### 1. 统一身份模型 `auth.Identity`

```go
type Identity struct {
    Provider    string
    UnionID     string
    OpenID      string
    PhoneNumber string
    ClientIP    string
}
```

业务侧只需要关心：

- `UnionID / OpenID`：第三方主键
- `PhoneNumber`：若授权拿到了手机号
- `ClientIP`：来源 IP

### 2. 业务方实现 `auth.UserHandler`

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

## 二、微信小程序登录 `auth/wechatmini`

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

- 只传 `code`：服务端先换取 `OpenID/UnionID`，若业务侧 `FindUser` 查不到用户，会先返回 `open_id`
- 同时传 `encrypted_data + iv_str`：服务端会尝试解密手机号，然后调用 `CreateUser`

### 4. 小程序码能力

除了登录路由，`wechatmini.Service` 还封了小程序码能力：

- `CreateQRCode(ctx, pagePath, width)`
- `GetCode(code MiniCode)`
- `GetUnlimitedCode(code *MiniUnlimitedCode)`

适合在商城分享、商品详情、邀请海报等场景直接复用。

```go
code := authwechatmini.NewMiniUnlimitedCode(ctx).
    SetPagePath("pages/goods/detail").
    SetScene("id=1001").
    SetWidth(430).
    SetEnvVersion("release")

resp, err := svc.GetUnlimitedCode(code)
```

## 三、推荐的 JWT 结合方式

认证子包最推荐的 token 签发方式，是统一复用 `wd.NewGinJWTMiddleware(...).TokenGenerator(...)`：

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

## 四、业务侧最佳实践

### 1. 查人顺序建议

对于微信小程序，推荐：

1. 先按 `UnionID` 查
2. 查不到再按 `OpenID` 查
3. 绑定后补齐 `UnionID`

因为 `UnionID` 更稳定，更适合做跨端统一账户。

### 2. `CreateUser` 不要做太多副作用

建议把 `CreateUser` 控制在“最小可注册闭环”内：

- 建会员主表
- 建第三方身份绑定表
- 补基础昵称/手机号

更重的逻辑如：

- 发欢迎券
- 发站内信
- 初始化资产账户

建议放异步任务或注册成功后的业务编排里。
