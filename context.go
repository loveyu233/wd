package wd

import (
	"context"
	"time"
)

// BackgroundContext 返回标准库的后台上下文，供仓库内统一复用。
func BackgroundContext() context.Context {
	return context.Background()
}

// BackgroundTimeout 基于后台上下文创建带超时的上下文。
func BackgroundTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(BackgroundContext(), timeout)
}

// Context 用来创建一个带超时的 context，默认 3 秒。
func Context(ttl ...int64) (context.Context, context.CancelFunc) {
	var sec int64 = 3
	if len(ttl) > 0 {
		sec = ttl[0]
	}
	return BackgroundTimeout(time.Second * time.Duration(sec))
}

// DurationSecond 用来把秒值转换为 time.Duration。
func DurationSecond(second int) time.Duration {
	return time.Duration(second) * time.Second
}
