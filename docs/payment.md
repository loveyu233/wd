# 支付模块

`wd/payment` 负责把“渠道支付 SDK 接入”与“业务订单逻辑”拆开。

当前内置：

- `payment/wechat`

它的目标不是替你决定订单系统怎么做，而是把这些重复工作统一掉：

- SDK 初始化
- 支付/退款路由注册
- 支付回调验签与解析
- 统一把渠道回调转成你自己的业务处理接口

## 一、统一设计思路

微信支付子包默认提供两种用法：

- 直接调用 `Pay / Refund / QueryOrder / QueryRefundOrder`
- 注册现成的 Gin 路由，直接接收标准请求体

如果你需要处理异步回调，再额外实现一个很小的 `Handler`：

```go
type Handler interface {
    OnPaymentNotify(ctx context.Context, notice PaymentNotify) error
    OnRefundNotify(ctx context.Context, notice RefundNotify) error
}
```

这意味着：

- 下单和退款不再要求你先实现 `BuildPayRequest / BuildRefundRequest`
- 工具库直接接收标准 `PayRequest / RefundRequest`
- 只有异步回调才需要业务方提供处理器

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
    yourWechatPayHandler, // 不需要回调时可直接传 nil
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

如果初始化时传了 `Handler`，还会注册：

- `POST /wechat/notify/payment`
- `POST /wechat/notify/refund`

### 3. 请求模型

金额统一使用“元”为单位的 `decimal.Decimal`，工具库内部会自动转换为微信支付要求的“分”：

```go
type PayRequest struct {
    Amount      decimal.Decimal `json:"amount"`
    Description string          `json:"description"`
    OpenID      string          `json:"openid"`
    Attach      string          `json:"attach"`
    NotifyURL   string          `json:"notify_url,omitempty"`
    OutTradeNo  string          `json:"out_trade_no"`
}

type RefundRequest struct {
    OrderID      string          `json:"order_id,omitempty"`
    TotalAmount  decimal.Decimal `json:"total_amount,omitempty"`
    RefundAmount decimal.Decimal `json:"refund_amount,omitempty"`
    RefundDesc   string          `json:"refund_desc,omitempty"`
    NotifyURL    string          `json:"notify_url,omitempty"`
}
```

要求：

- 金额必须大于 0
- 最多支持 2 位小数
- `RefundAmount` 不能大于 `TotalAmount`

### 4. 直接调用示例

```go
resp, err := svc.Pay(ctx, &paymentwechat.PayRequest{
    Amount:      decimal.RequireFromString("199.00"),
    Description: "年度会员",
    OpenID:      "openid-1001",
    OutTradeNo:  "order-20260410-0001",
})
```

退款：

```go
resp, err := svc.Refund(ctx, &paymentwechat.RefundRequest{
    OrderID:      "order-20260410-0001",
    TotalAmount:  decimal.RequireFromString("199.00"),
    RefundAmount: decimal.RequireFromString("199.00"),
    RefundDesc:   "用户申请退款",
})
```

### 5. 回调处理示例

```go
type payHandler struct{}

func (payHandler) OnPaymentNotify(ctx context.Context, notice paymentwechat.PaymentNotify) error {
    return markOrderPaid(notice.OrderID, notice.Attach)
}

func (payHandler) OnRefundNotify(ctx context.Context, notice paymentwechat.RefundNotify) error {
    return markOrderRefunded(notice.RefundOrderID)
}
```

### 6. 直接调用能力

除了走路由，你也可以直接用 service：

- `Pay(ctx, req)`
- `Refund(ctx, req)`
- `QueryOrder(ctx, orderID)`
- `QueryRefundOrder(ctx, orderID)`

适合：

- 内部 job 做支付状态补偿查询
- 管理后台手动退款
- 对账任务查询渠道订单状态

## 三、推荐的订单处理方式

### 1. 金额只信数据库，不信前端

前端最多传订单号，订单标题、金额、openid 都建议由业务系统自己查出来再组装 `PayRequest`。

### 2. 回调处理一定要幂等

推荐写法：

1. 根据 `orderID` 查订单
2. 如果订单已经是成功状态，直接返回成功
3. 如果状态未完成，则更新状态并记录渠道流水号
4. 整个过程最好放事务里

### 3. 工具库负责渠道，业务负责状态机

建议分层：

- `handler`：接 HTTP 请求，绑定参数
- `order service`：订单状态机、幂等、事务
- `payment/wechat`：渠道请求、验签、回调解析

## 四、仓库内可参考示例

- `test/projectflow/payment_flow_test.go`
- `test/projectflow/shared_test.go`
- `test/projectflow/wechatmini_real_flow_test.go`

其中 `wechatmini_real_flow_test.go` 展示了“小程序登录 -> 创建订单 -> 组装支付请求 -> 回调改状态”的完整业务链。
