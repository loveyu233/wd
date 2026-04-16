package wechat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	wechatpayment "github.com/ArtisanCloud/PowerWeChat/v3/src/payment"
	notifyrequest "github.com/ArtisanCloud/PowerWeChat/v3/src/payment/notify/request"
	orderrequest "github.com/ArtisanCloud/PowerWeChat/v3/src/payment/order/request"
	refundrequest "github.com/ArtisanCloud/PowerWeChat/v3/src/payment/refund/request"
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
	"github.com/shopspring/decimal"
)

var decimalYuanRatio = decimal.NewFromInt(100)

// Service 聚合微信支付的下单、退款与回调能力。
type Service struct {
	client         *wechatpayment.Payment
	handler        Handler
	saveHandlerLog bool
}

// New 用来根据配置初始化微信支付服务。
func New(config Config, handler Handler) (*Service, error) {
	if config.AppID == "" || config.MchID == "" {
		return nil, errors.New("微信支付 AppID 或 MchID 不能为空")
	}
	paymentApp, err := wechatpayment.NewPayment(&wechatpayment.UserConfig{
		AppID:              config.AppID,
		MchID:              config.MchID,
		MchApiV3Key:        config.MchApiV3Key,
		Key:                config.Key,
		CertPath:           config.CertPath,
		KeyPath:            config.KeyPath,
		SerialNo:           config.SerialNo,
		CertificateKeyPath: config.CertificateKeyPath,
		WechatPaySerial:    config.WechatPaySerial,
		RSAPublicKeyPath:   config.RSAPublicKeyPath,
		SubAppID:           config.SubAppID,
		SubMchID:           config.SubMchID,
		Http:               config.HTTP,
		ResponseType:       response.TYPE_MAP,
		Log:                config.Log,
		Cache:              config.Cache,
		HttpDebug:          config.HTTPDebug,
		NotifyURL:          config.NotifyURL,
	})
	if err != nil {
		return nil, err
	}
	return NewWithClient(paymentApp, handler, config.SaveHandlerLog)
}

// NewWithClient 用来复用外部传入的微信支付客户端。
// handler 可以为 nil，此时仍可直接调用 Pay/Refund，但不会注册异步回调路由。
func NewWithClient(client *wechatpayment.Payment, handler Handler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("微信支付客户端不能为空")
	}
	return &Service{client: client, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层微信支付客户端。
func (s *Service) Client() *wechatpayment.Payment {
	return s.client
}

func (s *Service) pay(c *gin.Context) {
	var payRequest PayRequest
	if err := c.ShouldBindJSON(&payRequest); err != nil {
		wd.ResponseParamError(c, err)
		return
	}
	resp, err := s.Pay(c.Request.Context(), &payRequest)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	wd.ResponseSuccess(c, resp)
}

func (s *Service) refund(c *gin.Context) {
	var refundRequest RefundRequest
	if err := c.ShouldBindJSON(&refundRequest); err != nil {
		wd.ResponseParamError(c, err)
		return
	}
	resp, err := s.Refund(c.Request.Context(), &refundRequest)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	wd.ResponseSuccess(c, resp)
}

func (s *Service) wxPayCallback(c *gin.Context) {
	res, err := s.payNotify(c.Request)
	if err != nil {
		writeCallbackFailure(c)
		return
	}
	if err = res.Write(c.Writer); err != nil {
		writeCallbackFailure(c)
	}
}

func (s *Service) wxRefundCallback(c *gin.Context) {
	res, err := s.refundNotify(c.Request)
	if err != nil {
		writeCallbackFailure(c)
		return
	}
	if err = res.Write(c.Writer); err != nil {
		writeCallbackFailure(c)
	}
}

// Pay 用来发起微信支付下单。
func (s *Service) Pay(ctx context.Context, req *PayRequest) (*PayResponse, error) {
	if err := validatePayRequest(req); err != nil {
		return nil, err
	}
	amountFen, err := convertPaymentAmountToFen(req.Amount, "amount")
	if err != nil {
		return nil, err
	}

	options := &orderrequest.RequestJSAPIPrepay{
		Amount:      &orderrequest.JSAPIAmount{Total: int(amountFen), Currency: "CNY"},
		Attach:      req.Attach,
		Description: req.Description,
		OutTradeNo:  req.OutTradeNo,
		Payer:       &orderrequest.JSAPIPayer{OpenID: req.OpenID},
	}
	if req.NotifyURL != "" {
		options.NotifyUrl = req.NotifyURL
	}
	resp, err := s.client.Order.JSAPITransaction(ctx, options)
	if err != nil {
		return nil, err
	}
	if resp.PrepayID == "" {
		return nil, errors.New("获取 prepay_id 失败")
	}
	payConf, err := s.client.JSSDK.BridgeConfig(resp.PrepayID, true)
	if err != nil {
		return nil, err
	}
	payload, ok := payConf.([]byte)
	if !ok {
		return nil, fmt.Errorf("微信支付桥接参数类型异常: %T", payConf)
	}
	var data PayResponse
	if err = json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	data.OutTradeNo = req.OutTradeNo
	return &data, nil
}

// Refund 用来发起微信支付退款。
func (s *Service) Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error) {
	if err := validateRefundRequest(req); err != nil {
		return nil, err
	}
	refundFen, err := convertPaymentAmountToFen(req.RefundAmount, "refund_amount")
	if err != nil {
		return nil, err
	}
	totalFen, err := convertPaymentAmountToFen(req.TotalAmount, "total_amount")
	if err != nil {
		return nil, err
	}

	outRefundNo := fmt.Sprintf("%s@%d", req.OrderID, wd.Now().Unix())
	options := &refundrequest.RequestRefund{
		OutTradeNo:  req.OrderID,
		OutRefundNo: outRefundNo,
		Reason:      req.RefundDesc,
		Amount: &refundrequest.RefundAmount{
			Refund:   int(refundFen),
			Total:    int(totalFen),
			From:     []*refundrequest.RefundAmountFrom{},
			Currency: "CNY",
		},
	}
	if req.NotifyURL != "" {
		options.NotifyUrl = req.NotifyURL
	}
	refundResp, err := s.client.Refund.Refund(ctx, options)
	if err != nil {
		return nil, err
	}
	if refundResp.TransactionID == "" {
		return nil, fmt.Errorf("获取退款交易号失败: %s", refundResp.Message)
	}
	return &RefundResponse{Code: 0, OutRefundNo: outRefundNo, Msg: "SUCCESS"}, nil
}

func (s *Service) payNotify(r *http.Request) (*http.Response, error) {
	if s.handler == nil {
		return nil, errors.New("未配置微信支付回调处理器")
	}
	ctx := r.Context()
	return s.client.HandlePaidNotify(r, func(message *notifyrequest.RequestNotify, transaction *models.Transaction, fail func(message string)) interface{} {
		if message.EventType != "TRANSACTION.SUCCESS" {
			return true
		}
		if transaction.OutTradeNo == "" {
			fail("payment fail")
			return false
		}
		notice := PaymentNotify{OrderID: transaction.OutTradeNo, Attach: transaction.Attach}
		if err := s.handler.OnPaymentNotify(ctx, notice); err != nil {
			fail("payment fail")
			return false
		}
		return true
	})
}

func (s *Service) refundNotify(r *http.Request) (*http.Response, error) {
	if s.handler == nil {
		return nil, errors.New("未配置微信支付回调处理器")
	}
	ctx := r.Context()
	return s.client.HandleRefundedNotify(r, func(message *notifyrequest.RequestNotify, transaction *models.Refund, fail func(message string)) interface{} {
		if message.EventType != "REFUND.SUCCESS" {
			return true
		}
		if transaction.OutRefundNo == "" {
			fail("refund fail")
			return false
		}
		notice := RefundNotify{RefundOrderID: transaction.OutRefundNo}
		if err := s.handler.OnRefundNotify(ctx, notice); err != nil {
			fail("refund fail")
			return false
		}
		return true
	})
}

// QueryOrder 用来查询微信支付订单。
func (s *Service) QueryOrder(ctx context.Context, orderID string) (*QueryOrderResponse, error) {
	return s.client.Order.QueryByOutTradeNumber(ctx, orderID)
}

// QueryRefundOrder 用来查询微信退款订单。
func (s *Service) QueryRefundOrder(ctx context.Context, orderID string) (*QueryRefundResponse, error) {
	return s.client.Refund.Query(ctx, orderID)
}

func validatePayRequest(req *PayRequest) error {
	if req == nil {
		return errors.New("支付请求不能为空")
	}
	if req.OutTradeNo == "" {
		return errors.New("out_trade_no 不能为空")
	}
	if req.Description == "" {
		return errors.New("description 不能为空")
	}
	if req.OpenID == "" {
		return errors.New("openid 不能为空")
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return errors.New("amount 必须大于 0")
	}
	return nil
}

func validateRefundRequest(req *RefundRequest) error {
	if req == nil {
		return errors.New("退款请求不能为空")
	}
	if req.OrderID == "" {
		return errors.New("order_id 不能为空")
	}
	if req.TotalAmount.LessThanOrEqual(decimal.Zero) {
		return errors.New("total_amount 必须大于 0")
	}
	if req.RefundAmount.LessThanOrEqual(decimal.Zero) {
		return errors.New("refund_amount 必须大于 0")
	}
	if req.RefundAmount.GreaterThan(req.TotalAmount) {
		return errors.New("refund_amount 不能大于 total_amount")
	}
	return nil
}

func convertPaymentAmountToFen(amount decimal.Decimal, fieldName string) (int64, error) {
	fen := amount.Mul(decimalYuanRatio)
	if !fen.Equal(fen.Truncate(0)) {
		return 0, fmt.Errorf("%s 最多支持 2 位小数", fieldName)
	}
	return fen.IntPart(), nil
}
