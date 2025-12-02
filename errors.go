package wd

import (
	"errors"

	"gorm.io/gorm"
)

// ErrRecordNotFound 用来判断错误是否表示记录不存在。
func ErrRecordNotFound(err error) bool {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	return false
}

// redisClientNilErr 用来生成 Redis 客户端未初始化的错误。
func redisClientNilErr() error {
	return errors.New("RedisClient为空,需要先使用gb.InitRedis()进行初始化")
}

// ErrDuplicatedKey 用来判断错误是否由唯一键冲突引起。
func ErrDuplicatedKey(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	return false
}

// ErrInvalidField 字段无效
func ErrInvalidField(err error) bool {
	if errors.Is(err, gorm.ErrInvalidField) {
		return true
	}
	return false
}

// ErrInvalidTransaction 数据库事务错误
func ErrInvalidTransaction(err error) bool {
	if errors.Is(err, gorm.ErrInvalidTransaction) {
		return true
	}
	return false
}
