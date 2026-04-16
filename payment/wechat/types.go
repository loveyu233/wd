package wechat

import (
	"context"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment/order/response"
	refundresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/payment/refund/response"
	"github.com/shopspring/decimal"
)

// Handler 约定业务方在开启回调路由时需要提供的通知处理逻辑。
type Handler interface {
	OnPaymentNotify(ctx context.Context, notice PaymentNotify) error
	OnRefundNotify(ctx context.Context, notice RefundNotify) error
}

// PayRequest 表示微信支付下单请求，金额统一使用元为单位的 decimal.Decimal。
type PayRequest struct {
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
	OpenID      string          `json:"openid"`
	Attach      string          `json:"attach"`
	NotifyURL   string          `json:"notify_url,omitempty"`
	OutTradeNo  string          `json:"out_trade_no"`
}

// PayResponse 表示返回给前端的小程序调起支付参数。
type PayResponse struct {
	AppID      string `json:"appId"`
	NonceStr   string `json:"nonceStr"`
	Package    string `json:"package"`
	PaySign    string `json:"paySign"`
	SignType   string `json:"signType"`
	TimeStamp  string `json:"timeStamp"`
	OutTradeNo string `json:"outTradeNo"`
	BizOrder   string `json:"bizOrder"`
}

// RefundRequest 表示微信退款请求，金额统一使用元为单位的 decimal.Decimal。
type RefundRequest struct {
	OrderID      string          `json:"order_id,omitempty"`
	TotalAmount  decimal.Decimal `json:"total_amount,omitempty"`
	RefundAmount decimal.Decimal `json:"refund_amount,omitempty"`
	RefundDesc   string          `json:"refund_desc,omitempty"`
	NotifyURL    string          `json:"notify_url,omitempty"`
}

// RefundResponse 表示退款发起结果。
type RefundResponse struct {
	Code        int    `json:"code"`
	OutRefundNo string `json:"out_refund_no"`
	Msg         string `json:"msg"`
}

// PaymentNotify 表示支付成功通知载荷。
type PaymentNotify struct {
	OrderID string
	Attach  string
}

// RefundNotify 表示退款通知载荷。
type RefundNotify struct {
	RefundOrderID string
}

// QueryOrderResponse 表示微信支付订单查询结果。
type QueryOrderResponse = response.ResponseOrder

// QueryRefundResponse 表示微信退款订单查询结果。
type QueryRefundResponse = refundresponse.ResponseRefund
