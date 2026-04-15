# 支付模块

`wd/payment` 负责把“渠道支付 SDK 接入”与“业务订单逻辑”拆开。

当前内置：

- `payment/wechat`
- `payment/alipay`

它的目标不是替你决定订单系统怎么做，而是把这些重复工作统一掉：

- SDK 初始化
- 支付/退款路由注册
- 支付回调验签与解析
- 统一把渠道回调转成你自己的业务处理接口

## 一、统一设计思路

支付模块强制业务方自己实现 `Handler`，这样非常适合商城、订单、收银台场景。

### 1. 微信支付 Handler

```go
type Handler interface {
    BuildPayRequest(c *gin.Context) (*PayRequest, error)
    BuildRefundRequest(c *gin.Context) (*RefundRequest, error)
    OnPaymentNotify(ctx context.Context, orderID, attach string) error
    OnRefundNotify(ctx context.Context, refundOrderID string) error
}
```

### 2. 支付宝 Handler

```go
type Handler interface {
    BuildPayRequest(c *gin.Context) (*PayRequest, error)
    BuildRefundRequest(c *gin.Context) (*RefundRequest, error)
    OnPaymentNotify(ctx context.Context, notice PaymentNotify) error
    OnRefundNotify(ctx context.Context, notice RefundNotify) error
}
```

你可以把它理解为两层分工：

- `wd/payment/*`：和第三方平台交互
- 你的 `Handler`：从订单系统拿业务数据，并决定订单状态怎么变更

## 二、微信支付 `payment/wechat`

### 1. 初始化

```go
svc, err := paymentwechat.New(
    paymentwechat.Config{
        AppID:          "wx-demo",
        MchID:          "mch-id",
        MchApiV3Key:    "api-v3-key",
        NotifyURL:      "https://api.example.com/payment/wechat/notify/payment",
        SaveHandlerLog: true,
    },
    yourWechatPayHandler,
)
```

如果项目里已经自己管理微信支付 SDK：

```go
svc, err := paymentwechat.NewWithClient(paymentClient, yourWechatPayHandler, true)
```

### 2. 路由注册

```go
payment.Register(r.Group("/wechat"), svc)
```

默认路由：

- `POST /wechat/pay`
- `POST /wechat/refund`
- `POST /wechat/notify/payment`
- `POST /wechat/notify/refund`

### 3. 业务侧下单请求构造

`BuildPayRequest` 的职责不是简单从前端抄字段，而是：

- 校验订单归属
- 校验订单状态是否允许支付
- 从数据库里拿真实金额
- 拼好 `Attach` 扩展字段

一个典型实现：

```go
func (h *payHandler) BuildPayRequest(c *gin.Context) (*paymentwechat.PayRequest, error) {
    var req struct {
        OrderNo string `json:"order_no" binding:"required"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        return nil, err
    }

    order := findOrderFromDB(req.OrderNo)
    if order == nil {
        return nil, errors.New("订单不存在")
    }
    if order.Status != "WAIT_PAY" {
        return nil, errors.New("当前订单不可支付")
    }

    return &paymentwechat.PayRequest{
        Price:       order.AmountFen,
        Description: order.Title,
        OpenID:      order.OpenID,
        Attach:      order.OrderNo,
        OutTradeNo:  order.OrderNo,
    }, nil
}
```

### 4. 直接调用能力

除了走路由，你也可以直接用 service：

- `Pay(ctx, req)`
- `Refund(ctx, req)`
- `QueryOrder(ctx, orderID)`
- `QueryRefundOrder(ctx, orderID)`

适合：

- 内部 job 做支付状态补偿查询
- 管理后台手动退款
- 对账任务查询渠道订单状态

## 三、支付宝支付 `payment/alipay`

### 1. 初始化

```go
svc, err := paymentalipay.New(
    paymentalipay.Config{
        AppID:                "2021xxxx",
        AppPrivateKey:        "private-key",
        AppPublicKeyFilePath: "./appPublicKey.pem",
        AliPublicKeyFilePath: "./alipayPublicKey_RSA2.pem",
        AliRootKeyFilePath:   "./alipayRootCert.crt",
        NotifyURL:            "https://api.example.com/payment/alipay/notify",
        SaveHandlerLog:       true,
    },
    yourAlipayHandler,
)
```

若已有客户端：

```go
svc, err := paymentalipay.NewWithClient(
    "app-id",
    alipayClient,
    aliPublicKey,
    "https://api.example.com/payment/alipay/notify",
    yourAlipayHandler,
    true,
)
```

### 2. 路由注册

```go
payment.Register(r.Group("/alipay"), svc)
```

默认路由：

- `POST /alipay/pay`
- `POST /alipay/refund`
- `POST /alipay/notify`

### 3. 直接调用能力

- `TradeCreate`
- `TradeQuery`
- `TradeRefund`
- `TradeFastPayRefundQuery`

与微信支付不同，支付宝回调会给你更完整的结构化载荷：

- `PaymentNotify`
- `RefundNotify`

业务侧通常只需要围绕 `OutTradeNo` 做幂等落库即可。

## 四、推荐的订单处理方式

### 1. 让支付模块只关心渠道，不关心订单表结构

也就是说：

- `payment/*` 模块不直接查你的订单表
- `Handler` 从你自己的 service/repo 查订单
- 模块只消费你组装好的支付请求数据

这样后续订单表怎么变都不会影响支付接入层。

### 2. 回调处理一定要幂等

不管微信还是支付宝，都必须假设回调会重试。

推荐写法：

1. 根据 `orderID/outTradeNo` 查订单
2. 如果订单已经是成功状态，直接返回成功
3. 如果状态未完成，则更新状态并记录渠道流水号
4. 整个过程最好放事务里

### 3. 金额只信数据库，不信前端

在 `BuildPayRequest` / `BuildRefundRequest` 里，前端最多传订单号，不要让前端直接决定金额。

### 4. 支付回调要和业务状态机保持一致

例如：

- `WAIT_PAY -> PAID`
- `PAID -> REFUNDING -> REFUNDED`

而不是回调一来就直接写任意状态。

## 五、常见接入分层

推荐把职责拆成这样：

- `handler`：接 HTTP 请求，解析参数
- `payment handler`：实现 `BuildPayRequest / BuildRefundRequest / OnNotify`
- `order service`：订单状态机、幂等、事务
- `repo`：订单表读写

这样支付接入升级、换渠道、补偿任务都比较容易维护。

## 六、仓库内可参考示例

- `test/projectflow/payment_flow_test.go`
- `test/projectflow/shared_test.go`
- `test/projectflow/wechatmini_real_flow_test.go`

其中 `wechatmini_real_flow_test.go` 最值得看，因为它把“小程序登录 -> 创建订单 -> 发起支付 -> 回调改状态”串成了一条完整业务链。
