package auth

import (
	"github.com/gin-gonic/gin"
	wd "github.com/loveyu233/wd"
	"github.com/spf13/cast"
)

// RespondLoginResult 用来把登录结果统一输出为 token 或结构化响应。
func RespondLoginResult(c *gin.Context, data any) {
	switch data.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		wd.ResponseSuccessToken(c, cast.ToString(data))
		return
	}
	wd.ResponseSuccess(c, data)
}
