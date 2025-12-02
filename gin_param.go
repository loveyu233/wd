package wd

import (
	"github.com/gin-gonic/gin"
)

// GinQueryDefault 用来获取 query 参数并在缺失时返回默认值。
func GinQueryDefault[T any](c *gin.Context, key string, defaultValue T) (T, error) {
	value := c.Query(key)

	// 如果参数为空，返回默认值
	if value == "" {
		return defaultValue, nil
	}

	// 转换为指定类型
	result, err := convertToType[T](value)
	if err != nil {
		// 返回可翻译的类型错误
		return defaultValue, CreateTypeError(key, value, err)
	}

	return result, nil
}

// GinQueryRequired 用来获取必须存在的 query 参数并转换类型。
func GinQueryRequired[T any](c *gin.Context, key string) (T, error) {
	var zero T
	value := c.Query(key)

	// 如果参数为空，返回 CreateRequiredError
	if value == "" {
		return zero, CreateRequiredError(key)
	}

	// 转换为指定类型
	result, err := convertToType[T](value)
	if err != nil {
		// 返回可翻译的类型错误
		return zero, CreateTypeError(key, value, err)
	}

	return result, nil
}

// GinPathRequired 用来读取路径参数并转换为指定类型。
func GinPathRequired[T any](c *gin.Context, key string) (T, error) {
	var zero T
	value := c.Param(key)

	// 如果参数为空，返回 CreateRequiredError
	if value == "" {
		return zero, CreateRequiredError(key)
	}

	// 转换为指定类型
	result, err := convertToType[T](value)
	if err != nil {
		// 返回可翻译的类型错误
		return zero, CreateTypeError(key, value, err)
	}

	return result, nil
}
