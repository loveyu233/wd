# 消息模块

`wd/message` 覆盖四类常见消息能力：

- `message/officialaccount`：微信公众号回调与主动消息
- `message/miniprogram`：微信小程序订阅消息
- `message/qywx`：企业微信机器人消息
- `message/sms`：阿里云短信

这四类能力虽然都叫“消息”，但使用方式并不一样：

- 公众号：有被动回调，也有主动推送
- 小程序订阅消息：通常是业务事件后主动发送
- 企业微信机器人：更偏告警、通知、运维机器人
- 短信：更偏验证码、营销通知、交易通知

## 一、统一入口

如果某个消息服务包含 Gin 路由，可以通过：

```go
message.Register(group, service)
```

当前带路由的主要是：

- `message/officialaccount`

而 `miniprogram`、`qywx`、`sms` 更适合在业务服务里直接注入并调用。

## 二、微信公众号 `message/officialaccount`

### 1. 适用场景

- 公众号回调验证
- 关注 / 取消关注事件处理
- 主动群发文本
- 模板消息发送

### 2. 业务方接口

```go
type Handler interface {
    OnSubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
    OnUnsubscribe(ctx context.Context, user *userresponse.ResponseGetUserInfo, event contract.EventInterface) error
    BuildPushRequest(c *gin.Context) (toUsers []string, message string, err error)
}
```

职责建议：

- `OnSubscribe`：记录关注、绑定渠道身份、发新手权益
- `OnUnsubscribe`：更新订阅状态、做流失标记
- `BuildPushRequest`：让主动推送接口参数与业务系统对齐

### 3. 初始化与路由

```go
svc, err := messageofficial.New(
    messageofficial.Config{
        AppID:          "wx-app-id",
        AppSecret:      "wx-secret",
        MessageToken:   "token",
        MessageAESKey:  "aes-key",
        SaveHandlerLog: true,
    },
    yourOfficialHandler,
)

message.Register(r.Group("/officialaccount"), svc)
```

默认路由：

- `GET /officialaccount/callback`
- `POST /officialaccount/callback`
- `POST /officialaccount/push`

### 4. 主动发送能力

- `Push(ctx, users, message)`：群发文本
- `PushTemplateMessage(ctx, openID, templateID, data)`：模板消息

模板消息示例：

```go
_, err = svc.PushTemplateMessage(ctx, "open-id", "template-id", struct {
    Title string `json:"title"`
    Time  string `json:"time"`
}{
    Title: "支付成功",
    Time:  "2026-04-15 10:00:00",
})
```

## 三、小程序订阅消息 `message/miniprogram`

### 1. 适用场景

适合在业务事件完成后主动发送，例如：

- 下单成功
- 支付成功
- 发货成功
- 审核通过/驳回
- 退款完成

### 2. 初始化

```go
svc, err := messageminiprogram.New(messageminiprogram.Config{
    AppID:  "wx-app-id",
    Secret: "secret",
})
```

### 3. 发送

```go
_, err = svc.SubscribeMessageSend(ctx, messageminiprogram.SubscribeContent{
    ToUserOpenID: "openid",
    TemplateID:   "template-id",
    Page:         "pages/order/detail?id=1",
    State:        messageminiprogram.StateFormal,
    Data: map[string]map[string]any{
        "thing1": {"value": "订单支付成功"},
        "time2":  {"value": "2026-04-15 10:00:00"},
    },
})
```

`State` 可选：

- `StateDeveloper`
- `StateTrial`
- `StateFormal`

## 四、企业微信机器人 `message/qywx`

### 1. 适用场景

这个模块更偏工程通知和告警：

- 发布结果通知
- 错误告警
- 库存告警
- 运营日报推送
- 文件报表发送

### 2. 初始化

```go
svc, err := messageqywx.New(messageqywx.Config{
    WebhookKey: "your-webhook-key",
})
```

### 3. 常用发送方式

文本：

```go
_, err = svc.SendText("服务已恢复")
```

Markdown：

```go
_, err = svc.SendMarkdown("**库存告警**\n> SKU-1001 库存不足")
```

文件：

```go
_, err = svc.SendFile("./report.xlsx", messageqywx.MediaTypeFile)
```

图文：

```go
_, err = svc.SendNews(
    messageqywx.Article{
        Title:       "日报已生成",
        Description: "点击查看完整报表",
        URL:         "https://example.com/report",
        PicURL:      "https://example.com/cover.png",
    },
)
```

### 4. 进阶能力

- `UploadMedia`
- `UploadMediaFromFile`
- `NewTextMessage(...).AddMention(...)`
- `NewTextMessage(...).AddMentionMobile(...)`

适合做“@某个人”或发送自定义上传文件的场景。

## 五、阿里云短信 `message/sms`

### 1. 适用场景

- 手机验证码
- 注册通知
- 支付成功通知
- 发货通知
- 营销批量发送

### 2. 初始化方式

通过 AccessKey：

```go
svc, err := messagesms.NewWithAccessKey("access-key-id", "access-key-secret")
```

通过完整凭证配置：

```go
svc, err := messagesms.New(messagesms.Config{
    CredentialConfig: new(credential.Config).
        SetType("access_key").
        SetAccessKeyId("id").
        SetAccessKeySecret("secret"),
})
```

复用已有客户端：

```go
svc, err := messagesms.NewWithClient(dysmsClient)
```

### 3. 发送方式

单条短信：

```go
err = svc.SendSimpleMsg(
    "13800138000",
    "签名名称",
    "SMS_123456789",
    `{"code":"1234"}`,
)
```

批量短信：

```go
err = svc.SendSimpleBatchMsg(
    `["13800138000","13900139000"]`,
    `["签名A","签名B"]`,
    "SMS_123456789",
    `[{"code":"1234"},{"code":"5678"}]`,
)
```

## 六、推荐的使用方式

### 1. 回调型消息和主动消息分层处理

- 公众号回调：走路由注册
- 小程序订阅消息 / 企业微信机器人 / 短信：在业务 service 里直接调用

### 2. 把消息发送当成副作用处理

业务上通常应该先保证主流程成功，再处理消息发送：

- 创建订单成功后再发订阅消息
- 支付成功后再发短信
- 发布成功后再发企业微信通知

必要时建议接入：

- 异步任务
- 重试机制
- 失败日志与补偿

### 3. 模板数据由业务层组装

尤其是：

- 公众号模板消息字段
- 小程序订阅消息字段
- 短信模板参数 JSON

这些最好都在业务层显式组装，避免消息模块和业务字段命名耦合。

## 七、仓库内可参考示例

- `test/projectflow/message_flow_test.go`
- `test/projectflow/shared_test.go`

如果你要找“怎么初始化”和“默认会注册哪些路由”，这两份测试最直接。
