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
}

// Error 返回包含错误码和提示信息的字符串。
func (e *AppError) Error() string {
	return fmt.Sprintf("错误码: %d, 错误信息: %s", e.Code, e.Message)
}

// WithMessage 创建一个携带自定义提示信息的新 AppError。
func (e *AppError) WithMessage(format string, args ...any) *AppError {
	if len(args) > 0 {
		format = fmt.Sprintf(format, args...)
	}
	if format == "" {
		format = e.Message
	}
	newErr := NewAppError(e.Code, format)
	return newErr
}

// NewAppError 根据错误码和消息生成 AppError。
func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// 预定义错误 http状态码 + 业务错误码
var (
	// 100xxx 请求外部服务失败
	ErrRequestExternalService = NewAppError(100000, "请求外部服务失败")
	ErrRequestWechat          = NewAppError(100001, "请求wechat服务失败")
	ErrRequestWechatPay       = NewAppError(100002, "请求wechat支付服务失败")
	ErrRequestAli             = NewAppError(100003, "请求zfb服务失败")
	ErrRequestAliPay          = NewAppError(100004, "请求zfb支付服务失败")

	// 400xxx 客户端错误
	ErrBadRequest         = NewAppError(400000, "请求错误")
	ErrInvalidParam       = NewAppError(400001, "请求参数错误")
	ErrTokenClientInvalid = NewAppError(400002, "token验证失败")
	ErrTokenServerInvalid = NewAppError(400003, "token生成失败")

	// 401xxx 未授权
	ErrUnauthorized = NewAppError(401000, "用户未登录或token已失效")

	// 403xxx 禁止操作
	ErrForbiddenAuth = NewAppError(403000, "权限不足")
	ErrUserDisabled  = NewAppError(403001, "用户不存在或已被禁用")

	// 404xxx 数据不存在
	ErrNotFound = NewAppError(404000, "数据不存在")

	// 409xxx 数据已存在
	ErrDataExists          = NewAppError(409000, "数据已存在")
	ErrUniqueIndexConflict = NewAppError(409001, "索引冲突")

	// 5xxxxx 服务器错误
	ErrServerBusy = NewAppError(500000, "服务器繁忙")
	ErrDatabase   = NewAppError(500001, "数据库错误")
	ErrRedis      = NewAppError(500002, "redis错误")

	ErrEncrypt = NewAppError(600000, "加密错误")

	ErrOther = NewAppError(999999, "其他错误")

	// ... 可以继续添加其他预定义错误
)

func MsgErrRequestExternalService(msg string, args ...any) *AppError {
	return ErrRequestExternalService.WithMessage(msg, args...)
}
func MsgErrRequestWechat(msg string, args ...any) *AppError {
	return ErrRequestWechat.WithMessage(msg, args...)
}
func MsgErrRequestWechatPay(msg string, args ...any) *AppError {
	return ErrRequestWechatPay.WithMessage(msg, args...)
}
func MsgErrRequestAli(msg string, args ...any) *AppError {
	return ErrRequestAli.WithMessage(msg, args...)
}
func MsgErrRequestAliPay(msg string, args ...any) *AppError {
	return ErrRequestAliPay.WithMessage(msg, args...)
}

func MsgErrBadRequest(msg string, args ...any) *AppError {
	return ErrBadRequest.WithMessage(msg, args...)
}
func MsgErrInvalidParam(msg string, args ...any) *AppError {
	return ErrInvalidParam.WithMessage(msg, args...)
}
func MsgErrTokenClientInvalid(msg string, args ...any) *AppError {
	return ErrTokenClientInvalid.WithMessage(msg, args...)
}
func MsgErrTokenServerInvalid(msg string, args ...any) *AppError {
	return ErrTokenServerInvalid.WithMessage(msg, args...)
}

func MsgErrUnauthorized(msg string, args ...any) *AppError {
	return ErrUnauthorized.WithMessage(msg, args...)
}

func MsgErrForbiddenAuth(msg string, args ...any) *AppError {
	return ErrForbiddenAuth.WithMessage(msg, args...)
}
func MsgErrUserDisabled(msg string, args ...any) *AppError {
	return ErrUserDisabled.WithMessage(msg, args...)
}

func MsgErrNotFound(msg string, args ...any) *AppError {
	return ErrNotFound.WithMessage(msg, args...)
}

func MsgErrDataExists(msg string, args ...any) *AppError {
	return ErrDataExists.WithMessage(msg, args...)
}
func MsgErrUniqueIndexConflict(msg string, args ...any) *AppError {
	return ErrUniqueIndexConflict.WithMessage(msg, args...)
}

func MsgErrServerBusy(msg string, args ...any) *AppError {
	return ErrServerBusy.WithMessage(msg, args...)
}
func MsgErrDatabase(msg string, args ...any) *AppError {
	return ErrDatabase.WithMessage(msg, args...)
}
func MsgErrRedis(msg string, args ...any) *AppError {
	return ErrRedis.WithMessage(msg, args...)
}

func MsgEncryptErr(msg string, args ...any) *AppError {
	return ErrEncrypt.WithMessage(msg, args...)
}

func ErrIsAppErr(err error, appErr *AppError) bool {
	return ConvertToAppError(err).Code == appErr.Code
}

// ReturnErrDatabase 将数据库错误映射成业务错误并处理未找到情况。
func ReturnErrDatabase(err error, msg string, notfoundMsg ...string) *AppError {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if len(notfoundMsg) == 0 {
			notfoundMsg = append(notfoundMsg, ErrNotFound.Message)
		}
		return ErrNotFound.WithMessage(notfoundMsg[0])
	}
	return ErrDatabase.WithMessage(msg)
}

// ConvertToAppError 把任意错误转换成统一的业务错误模型。
func ConvertToAppError(err error) *AppError {
	if err == nil {
		return ErrServerBusy.WithMessage("未知错误")
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Code == ErrInvalidParam.Code {
			appErr.Message = TranslateError(errors.New(appErr.Message)).Error()
			return appErr
		}
		return appErr
	}

	// 映射特定的错误到业务错误
	switch {
	case ErrRecordNotFound(err):
		return ErrNotFound.WithMessage("数据不存在")
	case ErrDuplicatedKey(err):
		return ErrDataExists.WithMessage("数据冲突")
	case ErrInvalidField(err):
		return ErrDatabase.WithMessage("字段无效")
	case ErrInvalidTransaction(err):
		return ErrDatabase.WithMessage("数据库事务错误")
	}

	// 处理mysql特定错误
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return ErrDatabase.WithMessage(mysqlErr.Message)
	}

	return ErrOther.WithMessage(err.Error())
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
	c.Set(CUSTOMCONSTRESPSTATUS, appErr.Code)
	c.Set(CUSTOMCONSTRESPMSG, appErr.Message)
	c.JSON(http.StatusOK, &Response{
		Code:    appErr.Code,
		Message: appErr.Message,
	})
}

// ResponseParamError 输出校验失败时的 JSON 响应。
func ResponseParamError(c *gin.Context, err error) {
	GetContextLogger(c).Error().Msg("resp_err", err.Error())
	te := TranslateError(err).Error()
	c.Set(CUSTOMCONSTRESPSTATUS, ErrInvalidParam.Code)
	c.Set(CUSTOMCONSTRESPMSG, te)
	if te == "" {
		te = ErrInvalidParam.Message
	}
	c.JSON(http.StatusOK, &Response{
		Code:    ErrInvalidParam.Code,
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
	c.Set(CUSTOMCONSTRESPSTATUS, http.StatusOK)
	c.Set(CUSTOMCONSTRESPMSG, message)
	c.JSON(http.StatusOK, &Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	})
}

// ResponseSuccessMsg 只返回成功的msg没有data
func ResponseSuccessMsg(c *gin.Context, msg string) {
	c.Set(CUSTOMCONSTRESPSTATUS, http.StatusOK)
	c.Set(CUSTOMCONSTRESPMSG, msg)
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
	c.Set(CUSTOMCONSTRESPSTATUS, http.StatusOK)
	c.Set(CUSTOMCONSTRESPMSG, "请求成功")
	response, err := EncryptData(data, custom)
	if err != nil {
		c.JSON(http.StatusOK, &Response{
			Code:    ErrEncrypt.Code,
			Message: ErrEncrypt.Message,
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
