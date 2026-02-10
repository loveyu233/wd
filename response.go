package wd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// AppError 自定义错误类型
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	E       error  `json:"e,omitempty"`
}

// Error 返回包含错误码和提示信息的字符串。
func (e *AppError) Error() string {
	return fmt.Sprintf("错误码: %d, 错误信息: %s", e.Code, e.Message)
}

// WithMessage 创建一个携带自定义提示信息的新 AppError。
func (e *AppError) WithMessage(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = e.Message
	}
	var newErr *AppError
	if len(errs) > 0 {
		newErr = NewAppError(e.Code, msg, e)
	} else {
		newErr = NewAppError(e.Code, msg, nil)
	}
	return newErr
}

// NewAppError 根据错误码和消息生成 AppError。
func NewAppError(code int, message string, e error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		E:       e,
	}
}

// 预定义错误 http状态码 + 业务错误码
var (
	// 100xxx 请求外部服务失败
	errRequestExternalService = NewAppError(100000, "服务请求失败，请稍后重试", nil)
	errRequestWechat          = NewAppError(100001, "微信服务请求失败", nil)
	errRequestWechatPay       = NewAppError(100002, "微信支付请求失败", nil)
	errRequestAli             = NewAppError(100003, "支付宝服务请求失败", nil)
	errRequestAliPay          = NewAppError(100004, "支付宝支付请求失败", nil)

	// 400xxx 客户端错误
	errBadRequest         = NewAppError(400000, "请求错误", nil)
	errInvalidParam       = NewAppError(400001, "请求参数错误", nil)
	errTokenClientInvalid = NewAppError(400002, "登陆凭证无效", nil)
	errTokenServerInvalid = NewAppError(400003, "登陆凭证生成失败", nil)

	// 401xxx 未授权
	errUnauthorized = NewAppError(401000, "请先登录", nil)

	// 403xxx 禁止操作
	errForbiddenAuth = NewAppError(403000, "权限不足", nil)
	errUserDisabled  = NewAppError(403001, "用户不存在或已被禁用", nil)

	// 404xxx 数据不存在
	errNotFound = NewAppError(404000, "数据不存在", nil)

	// 409xxx 数据已存在
	errDataExists          = NewAppError(409000, "数据已存在", nil)
	errUniqueIndexConflict = NewAppError(409001, "数据已存在", nil)

	// 5xxxxx 服务器错误
	errServerBusy = NewAppError(500000, "服务繁忙，请稍后重试", nil)
	errDatabase   = NewAppError(500001, "服务异常，请稍后重试", nil)
	errRedis      = NewAppError(500002, "服务异常，请稍后重试", nil)

	errEncrypt = NewAppError(600000, "数据处理失败", nil)

	errOther = NewAppError(999999, "操作失败，请稍后重试", nil)

	// ... 可以继续添加其他预定义错误
)

func RespCodeDescMap() map[int]string {
	return map[int]string{
		200:    "请求成功",
		100000: "服务请求失败，请稍后重试",
		100001: "微信服务请求失败",
		100002: "微信支付请求失败",
		100003: "支付宝服务请求失败",
		100004: "支付宝支付请求失败",
		400000: "请求错误",
		400001: "请求参数错误",
		400002: "登陆凭证无效",
		400003: "登陆凭证生成失败",
		401000: "请先登录",
		403000: "权限不足",
		403001: "用户不存在或已被禁用",
		404000: "数据不存在",
		409000: "数据已存在",
		409001: "数据已存在",
		500000: "服务繁忙，请稍后重试",
		500001: "服务异常，请稍后重试",
		500002: "服务异常，请稍后重试",
		600000: "数据处理失败",
		999999: "操作失败，请稍后重试",
	}
}

func MsgErrRequestExternalService(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRequestExternalService.Message
	}
	return errRequestExternalService.WithMessage(msg, errs...)
}
func MsgErrRequestWechat(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRequestWechat.Message
	}
	return errRequestWechat.WithMessage(msg, errs...)
}
func MsgErrRequestWechatPay(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRequestWechatPay.Message
	}
	return errRequestWechatPay.WithMessage(msg, errs...)
}
func MsgErrRequestAli(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRequestAli.Message
	}
	return errRequestAli.WithMessage(msg, errs...)
}
func MsgErrRequestAliPay(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRequestAliPay.Message
	}
	return errRequestAliPay.WithMessage(msg, errs...)
}

func MsgErrBadRequest(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errBadRequest.Message
	}
	return errBadRequest.WithMessage(msg, errs...)
}
func MsgErrInvalidParam(err error) *AppError {
	return errInvalidParam.WithMessage(TranslateError(err).Error())
}
func MsgErrTokenClientInvalid(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errTokenClientInvalid.Message
	}
	return errTokenClientInvalid.WithMessage(msg, errs...)
}
func MsgErrTokenServerInvalid(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errTokenServerInvalid.Message
	}
	return errTokenServerInvalid.WithMessage(msg, errs...)
}

func MsgErrUnauthorized(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errUnauthorized.Message
	}
	return errUnauthorized.WithMessage(msg, errs...)
}

func MsgErrForbiddenAuth(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errForbiddenAuth.Message
	}
	return errForbiddenAuth.WithMessage(msg, errs...)
}
func MsgErrUserDisabled(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errUserDisabled.Message
	}
	return errUserDisabled.WithMessage(msg, errs...)
}

func MsgErrNotFound(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errNotFound.Message
	}
	return errNotFound.WithMessage(msg, errs...)
}

func MsgErrDataExists(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errDataExists.Message
	}
	return errDataExists.WithMessage(msg, errs...)
}
func MsgErrUniqueIndexConflict(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errUniqueIndexConflict.Message
	}
	return errUniqueIndexConflict.WithMessage(msg, errs...)
}

func MsgErrServerBusy(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errServerBusy.Message
	}
	return errServerBusy.WithMessage(msg, errs...)
}
func MsgErrDatabase(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errDatabase.Message
	}
	return errDatabase.WithMessage(msg, errs...)
}
func MsgErrRedis(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errRedis.Message
	}
	return errRedis.WithMessage(msg, errs...)
}

func MsgEncryptErr(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errEncrypt.Message
	}
	return errEncrypt.WithMessage(msg, errs...)
}

func MsgErrOther(msg string, errs ...error) *AppError {
	if msg == "" {
		msg = errOther.Message
	}
	return errOther.WithMessage(msg, errs...)
}

func ErrIsAppErr(err error, appErr *AppError) bool {
	return ConvertToAppError(err).Code == appErr.Code
}

// ReturnErrDatabase 将数据库错误映射成业务错误并处理未找到情况。
func ReturnErrDatabase(err error, msg string, notfoundMsg ...string) *AppError {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if len(notfoundMsg) == 0 {
			notfoundMsg = append(notfoundMsg, errNotFound.Message)
		}
		return MsgErrNotFound(notfoundMsg[0], err)
	}
	return MsgErrDatabase(msg, err)
}

// ConvertToAppError 把任意错误转换成统一的业务错误模型。
func ConvertToAppError(err error) *AppError {
	if err == nil {
		return MsgErrServerBusy("服务异常，请稍后重试")
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Code == errInvalidParam.Code {
			appErr.Message = TranslateError(errors.New(appErr.Message)).Error()
			return appErr
		}
		return appErr
	}

	// 映射特定的错误到业务错误
	switch {
	case ErrRecordNotFound(err):
		return MsgErrNotFound("数据不存在", err)
	case ErrDuplicatedKey(err):
		return MsgErrDataExists("数据已存在", err)
	case ErrInvalidField(err):
		return MsgErrDatabase("数据处理失败，请检查输入", err)
	case ErrInvalidTransaction(err):
		return MsgErrDatabase("服务异常，请稍后重试", err)
	}

	// 处理mysql特定错误
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return MsgErrDatabase("服务异常，请稍后重试", err)
	}

	return MsgErrOther("操作失败，请稍后重试", err)
}

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// ResponseError 根据错误输出统一的 JSON 响应。
func ResponseError(c *gin.Context, err error) {
	GetContextLogger(c).Error().Msg("resp_err", err.Error())
	appErr := ConvertToAppError(err)
	c.Set(CtxKeyRespStatus, appErr.Code)
	c.Set(CtxKeyRespMsg, appErr.Message)
	c.JSON(http.StatusOK, &Response{
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}

// ResponseParamError 输出校验失败时的 JSON 响应。
func ResponseParamError(c *gin.Context, err error) {
	GetContextLogger(c).Error().Msg("resp_err", err.Error())
	te := TranslateError(err).Error()
	c.Set(CtxKeyRespStatus, errInvalidParam.Code)
	c.Set(CtxKeyRespMsg, te)
	if te == "" {
		te = errInvalidParam.Message
	}
	c.JSON(http.StatusOK, &Response{
		Code:    errInvalidParam.Code,
		Message: te,
	})
}

// ResponseSuccess 返回包含数据的成功响应。如果是非对象时，将数据转为字符串在message中返回
func ResponseSuccess(c *gin.Context, data any, msg ...string) {
	var message = "操作成功"
	if len(msg) > 0 {
		message = msg[0]
	}
	switch data.(type) {
	case string, int, int8, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64:
		ResponseSuccessMsg(c, cast.ToString(data))
		return
	}
	c.Set(CtxKeyRespStatus, http.StatusOK)
	c.Set(CtxKeyRespMsg, message)
	c.JSON(http.StatusOK, &Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	})
}

// ResponseSuccessMsg 只返回成功的msg没有data
func ResponseSuccessMsg(c *gin.Context, msg string) {
	c.Set(CtxKeyRespStatus, http.StatusOK)
	c.Set(CtxKeyRespMsg, msg)
	c.JSON(http.StatusOK, &Response{
		Code:    http.StatusOK,
		Message: msg,
	})
}

// ResponseSuccessToken 返回token使用
func ResponseSuccessToken(c *gin.Context, token string) {
	ResponseSuccess(c, gin.H{
		"token": token,
	})
}

// ResponseSuccessEncryptData 对响应数据进行加密后返回。
func ResponseSuccessEncryptData(c *gin.Context, data any, custom func(now int64) (key, nonce string)) {
	c.Set(CtxKeyRespStatus, http.StatusOK)
	c.Set(CtxKeyRespMsg, "请求成功")
	response, err := EncryptData(data, custom)
	if err != nil {
		c.JSON(http.StatusOK, &Response{
			Code:    errEncrypt.Code,
			Message: errEncrypt.Message,
		})
		return
	}
	c.JSON(http.StatusOK, &Response{
		Code:    http.StatusOK,
		Message: "请求成功",
		Data:    response,
	})
}

// ResponseThirdPartyHTTPBody 直接转发第三方响应体和状态码。
func ResponseThirdPartyHTTPBody(c *gin.Context, body any, code ...int) {
	if len(code) == 0 {
		code = append(code, 200)
	}
	c.JSON(code[0], body)
}

func ReturnAppErr(fn func() error) error {
	if err := fn(); err != nil {
		return ConvertToAppError(err)
	}
	return nil
}
