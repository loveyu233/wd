package alipay

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	alipayv3 "github.com/go-pay/gopay/alipay/v3"
)

// Handler 约定业务方需要提供的支付宝支付逻辑。
type Handler interface {
	BuildPayRequest(c *gin.Context) (*PayRequest, error)
	BuildRefundRequest(c *gin.Context) (*RefundRequest, error)
	OnPaymentNotify(ctx context.Context, notice PaymentNotify) error
	OnRefundNotify(ctx context.Context, notice RefundNotify) error
}

// PayRequest 表示支付宝支付请求。
type PayRequest struct {
	OutTradeNo  string
	BuyerOpenID string
	Subject     string
	TotalAmount float64
}

// RefundRequest 表示支付宝退款请求。
type RefundRequest struct {
	OutTradeNo   string
	RefundReason string
	RefundAmount float64
}

// NotifyPayload 表示支付宝异步通知的原始解析结果。
type NotifyPayload struct {
	AppID          string    `json:"app_id"`
	NotifyTime     time.Time `json:"notify_time"`
	TradeNo        string    `json:"trade_no"`
	OutTradeNo     string    `json:"out_trade_no"`
	OutBizNo       string    `json:"out_biz_no"`
	BuyerOpenID    string    `json:"buyer_open_id"`
	BuyerLogonID   string    `json:"buyer_logon_id"`
	TradeStatus    string    `json:"trade_status"`
	TotalAmount    float64   `json:"total_amount"`
	ReceiptAmount  float64   `json:"receipt_amount"`
	BuyerPayAmount float64   `json:"buyer_pay_amount"`
	PointAmount    float64   `json:"point_amount"`
	RefundFee      float64   `json:"refund_fee"`
	SendBackFee    float64   `json:"send_back_fee"`
	Subject        string    `json:"subject"`
	PassbackParams string    `json:"passback_params"`
	GmtCreate      time.Time `json:"gmt_create"`
	GmtPayment     time.Time `json:"gmt_payment"`
	GmtRefund      time.Time `json:"gmt_refund"`
	GmtClose       time.Time `json:"gmt_close"`
}

// PaymentNotify 表示支付成功通知载荷。
type PaymentNotify struct {
	TradeNo        string
	OutTradeNo     string
	BuyerOpenID    string
	BuyerLogonID   string
	TradeStatus    string
	TotalAmount    float64
	ReceiptAmount  float64
	BuyerPayAmount float64
	PassbackParams string
	Subject        string
	NotifyTime     time.Time
}

// RefundNotify 表示退款通知载荷。
type RefundNotify struct {
	TradeNo        string
	OutTradeNo     string
	RefundFee      float64
	SendBackFee    float64
	PassbackParams string
	NotifyTime     time.Time
}

// TradeCreateResponse 表示支付宝支付下单响应。
type TradeCreateResponse = alipayv3.TradeCreateRsp

// TradeRefundResponse 表示支付宝退款响应。
type TradeRefundResponse = alipayv3.TradeRefundRsp

// TradeQueryResponse 表示支付宝订单查询响应。
type TradeQueryResponse = alipayv3.TradeQueryRsp

// RefundQueryResponse 表示支付宝退款查询响应。
type RefundQueryResponse = alipayv3.TradeFastPayRefundQueryRsp
