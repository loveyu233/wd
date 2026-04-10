package wechat

import (
	"context"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/payment/order/response"
	refundresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/payment/refund/response"
	"github.com/gin-gonic/gin"
)

// Handler 约定业务方需要提供的微信支付请求组装与回调处理逻辑。
type Handler interface {
	BuildPayRequest(c *gin.Context) (*PayRequest, error)
	BuildRefundRequest(c *gin.Context) (*RefundRequest, error)
	OnPaymentNotify(ctx context.Context, orderID, attach string) error
	OnRefundNotify(ctx context.Context, refundOrderID string) error
}

// PayRequest 表示微信支付下单请求。
type PayRequest struct {
	Price       int64  `json:"price"`
	Description string `json:"description"`
	IP          string `json:"ip,omitempty"`
	OpenID      string `json:"openid"`
	Attach      string `json:"attach"`
	NotifyURL   string `json:"notify_url,omitempty"`
	OutTradeNo  string `json:"out_trade_no"`
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

// RefundRequest 表示微信退款请求。
type RefundRequest struct {
	OrderID    string `json:"order_id,omitempty"`
	TotalFee   int    `json:"total_fee,omitempty"`
	RefundFee  int    `json:"refund_fee,omitempty"`
	RefundDesc string `json:"refund_desc,omitempty"`
	NotifyURL  string `json:"notify_url,omitempty"`
}

// RefundResponse 表示退款发起结果。
type RefundResponse struct {
	Code        int    `json:"code"`
	OutRefundNo string `json:"out_refund_no"`
	Msg         string `json:"msg"`
}

// QueryOrderResponse 表示微信支付订单查询结果。
type QueryOrderResponse = response.ResponseOrder

// QueryRefundResponse 表示微信退款订单查询结果。
type QueryRefundResponse = refundresponse.ResponseRefund
