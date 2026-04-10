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
	"github.com/loveyu233/wd/internal/xhelper"
)

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
	if xhelper.IsNil(handler) {
		return nil, errors.New("微信支付处理器不能为空")
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
func NewWithClient(client *wechatpayment.Payment, handler Handler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("微信支付客户端不能为空")
	}
	if xhelper.IsNil(handler) {
		return nil, errors.New("微信支付处理器不能为空")
	}
	return &Service{client: client, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层微信支付客户端。
func (s *Service) Client() *wechatpayment.Payment {
	return s.client
}

func (s *Service) pay(c *gin.Context) {
	payRequest, err := s.handler.BuildPayRequest(c)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	resp, err := s.Pay(c.Request.Context(), payRequest)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	wd.ResponseSuccess(c, resp)
}

func (s *Service) refund(c *gin.Context) {
	refundRequest, err := s.handler.BuildRefundRequest(c)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	resp, err := s.Refund(c.Request.Context(), refundRequest)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestWechatPay("微信支付请求失败", err))
		return
	}
	wd.ResponseSuccess(c, resp)
}

func (s *Service) wxPayCallback(c *gin.Context) {
	res, err := s.payNotify(c.Request, s.handler.OnPaymentNotify)
	if err != nil {
		writeCallbackFailure(c)
		return
	}
	if err = res.Write(c.Writer); err != nil {
		writeCallbackFailure(c)
	}
}

func (s *Service) wxRefundCallback(c *gin.Context) {
	res, err := s.refundNotify(c.Request, s.handler.OnRefundNotify)
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
	if req == nil {
		return nil, errors.New("支付请求不能为空")
	}
	options := &orderrequest.RequestJSAPIPrepay{
		Amount:      &orderrequest.JSAPIAmount{Total: int(req.Price), Currency: "CNY"},
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
	if req == nil {
		return nil, errors.New("退款请求不能为空")
	}
	outRefundNo := fmt.Sprintf("%s@%d", req.OrderID, wd.Now().Unix())
	options := &refundrequest.RequestRefund{
		OutTradeNo:  req.OrderID,
		OutRefundNo: outRefundNo,
		Reason:      req.RefundDesc,
		Amount: &refundrequest.RefundAmount{
			Refund:   req.RefundFee,
			Total:    req.TotalFee,
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

func (s *Service) payNotify(r *http.Request, callback func(ctx context.Context, orderID, attach string) error) (*http.Response, error) {
	ctx := r.Context()
	return s.client.HandlePaidNotify(r, func(message *notifyrequest.RequestNotify, transaction *models.Transaction, fail func(message string)) interface{} {
		if message.EventType != "TRANSACTION.SUCCESS" {
			return true
		}
		if transaction.OutTradeNo == "" {
			fail("payment fail")
			return false
		}
		if err := callback(ctx, transaction.OutTradeNo, transaction.Attach); err != nil {
			fail("payment fail")
			return false
		}
		return true
	})
}

func (s *Service) refundNotify(r *http.Request, callback func(ctx context.Context, refundOrderID string) error) (*http.Response, error) {
	ctx := r.Context()
	return s.client.HandleRefundedNotify(r, func(message *notifyrequest.RequestNotify, transaction *models.Refund, fail func(message string)) interface{} {
		if message.EventType != "REFUND.SUCCESS" {
			return true
		}
		if transaction.OutRefundNo == "" {
			fail("refund fail")
			return false
		}
		if err := callback(ctx, transaction.OutRefundNo); err != nil {
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
