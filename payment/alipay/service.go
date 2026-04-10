package alipay

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	alipayv2 "github.com/go-pay/gopay/alipay"
	alipayv3 "github.com/go-pay/gopay/alipay/v3"
	wd "github.com/loveyu233/wd"
	"github.com/loveyu233/wd/internal/xclients"
	"github.com/loveyu233/wd/internal/xhelper"
)

// Service 聚合支付宝支付与回调处理能力。
type Service struct {
	appID          string
	client         *alipayv3.ClientV3
	aliPublicKey   []byte
	notifyURL      string
	handler        Handler
	saveHandlerLog bool
}

// New 用来根据配置初始化支付宝支付服务。
func New(config Config, handler Handler) (*Service, error) {
	if xhelper.IsNil(handler) {
		return nil, errors.New("支付宝支付处理器不能为空")
	}
	bundle, err := xclients.NewAlipayBundle(xclients.AlipayConfig{
		AppID:                config.AppID,
		AppPrivateKey:        config.AppPrivateKey,
		AESKey:               config.AESKey,
		AppPublicKeyFilePath: config.AppPublicKeyFilePath,
		AliPublicKeyFilePath: config.AliPublicKeyFilePath,
		AliRootKeyFilePath:   config.AliRootKeyFilePath,
	})
	if err != nil {
		return nil, err
	}
	return &Service{appID: config.AppID, client: bundle.Client, aliPublicKey: bundle.AliPublicKey, notifyURL: config.NotifyURL, handler: handler, saveHandlerLog: config.SaveHandlerLog}, nil
}

// NewWithClient 用来复用调用方传入的支付宝客户端。
func NewWithClient(appID string, client *alipayv3.ClientV3, aliPublicKey []byte, notifyURL string, handler Handler, saveHandlerLog bool) (*Service, error) {
	if client == nil {
		return nil, errors.New("支付宝客户端不能为空")
	}
	if appID == "" {
		return nil, errors.New("支付宝 AppID 不能为空")
	}
	if len(aliPublicKey) == 0 {
		return nil, errors.New("支付宝公钥不能为空")
	}
	if xhelper.IsNil(handler) {
		return nil, errors.New("支付宝支付处理器不能为空")
	}
	return &Service{appID: appID, client: client, aliPublicKey: aliPublicKey, notifyURL: notifyURL, handler: handler, saveHandlerLog: saveHandlerLog}, nil
}

// Client 用来返回底层支付宝客户端。
func (s *Service) Client() *alipayv3.ClientV3 {
	return s.client
}

func (s *Service) pay(c *gin.Context) {
	param, err := s.handler.BuildPayRequest(c)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝支付请求失败", err))
		return
	}
	resp, err := s.TradeCreate(c.Request.Context(), param)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝支付请求失败", err))
		return
	}
	if resp.StatusCode != 10000 {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝支付请求失败", errors.New(resp.ErrResponse.Message)))
		return
	}
	wd.ResponseSuccess(c, gin.H{"trade_no": resp.TradeNo, "out_trade_no": resp.OutTradeNo})
}

func (s *Service) refund(c *gin.Context) {
	param, err := s.handler.BuildRefundRequest(c)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝退款请求失败", err))
		return
	}
	resp, err := s.TradeRefund(c.Request.Context(), param)
	if err != nil {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝退款请求失败", err))
		return
	}
	if resp.StatusCode != 10000 {
		wd.ResponseError(c, wd.MsgErrRequestAliPay("支付宝退款请求失败", errors.New(resp.ErrResponse.Message)))
		return
	}
	wd.ResponseSuccess(c, gin.H{"trade_no": resp.TradeNo, "out_trade_no": resp.OutTradeNo, "refund_fee": resp.RefundFee})
}

func (s *Service) notify(c *gin.Context) {
	payload, err := s.parseNotify(c.Request)
	if err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	ctx := c.Request.Context()
	if payload.RefundFee != 0 {
		err = s.handler.OnRefundNotify(ctx, RefundNotify{
			TradeNo:        payload.TradeNo,
			OutTradeNo:     payload.OutTradeNo,
			RefundFee:      payload.RefundFee,
			SendBackFee:    payload.SendBackFee,
			PassbackParams: payload.PassbackParams,
			NotifyTime:     payload.NotifyTime,
		})
	} else if payload.ReceiptAmount != 0 {
		err = s.handler.OnPaymentNotify(ctx, PaymentNotify{
			TradeNo:        payload.TradeNo,
			OutTradeNo:     payload.OutTradeNo,
			BuyerOpenID:    payload.BuyerOpenID,
			BuyerLogonID:   payload.BuyerLogonID,
			TradeStatus:    payload.TradeStatus,
			TotalAmount:    payload.TotalAmount,
			ReceiptAmount:  payload.ReceiptAmount,
			BuyerPayAmount: payload.BuyerPayAmount,
			PassbackParams: payload.PassbackParams,
			Subject:        payload.Subject,
			NotifyTime:     payload.NotifyTime,
		})
	}
	if err != nil {
		c.String(http.StatusOK, "fail")
		return
	}
	c.String(http.StatusOK, "success")
}

// TradeCreate 用来发起支付宝支付下单。
func (s *Service) TradeCreate(ctx context.Context, param *PayRequest) (*TradeCreateResponse, error) {
	if param == nil {
		return nil, errors.New("支付请求不能为空")
	}
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", param.OutTradeNo).
		Set("total_amount", param.TotalAmount).
		Set("product_code", "JSAPI_PAY").
		Set("op_app_id", s.appID).
		Set("buyer_open_id", param.BuyerOpenID).
		Set("subject", param.Subject)
	if s.notifyURL != "" {
		bm.Set("notify_url", s.notifyURL)
	}
	return s.client.TradeCreate(ctx, bm)
}

// TradeQuery 用来查询支付宝订单。
func (s *Service) TradeQuery(ctx context.Context, outTradeNo string) (*TradeQueryResponse, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", outTradeNo)
	return s.client.TradeQuery(ctx, bm)
}

// TradeRefund 用来发起支付宝退款。
func (s *Service) TradeRefund(ctx context.Context, param *RefundRequest) (*TradeRefundResponse, error) {
	if param == nil {
		return nil, errors.New("退款请求不能为空")
	}
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", param.OutTradeNo).
		Set("refund_amount", param.RefundAmount).
		Set("refund_reason", param.RefundReason)
	return s.client.TradeRefund(ctx, bm)
}

// TradeFastPayRefundQuery 用来查询支付宝退款结果。
func (s *Service) TradeFastPayRefundQuery(ctx context.Context, outTradeNo, outRequestNo string) (*RefundQueryResponse, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", outTradeNo).Set("out_request_no", outRequestNo)
	return s.client.TradeFastPayRefundQuery(ctx, bm)
}

func (s *Service) parseNotify(req *http.Request) (*NotifyPayload, error) {
	bodyMap, err := alipayv2.ParseNotifyToBodyMap(req)
	if err != nil {
		return nil, err
	}
	ok, err := alipayv2.VerifySign(string(s.aliPublicKey), bodyMap)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("支付宝回调签名校验失败")
	}
	var payload NotifyPayload
	if err := bodyMap.Unmarshal(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}
