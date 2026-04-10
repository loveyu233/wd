package qywx

import (
	"net/http"
	"time"
)

// Config 用来描述企业微信机器人消息服务的配置。
type Config struct {
	WebhookKey string
	HTTPClient *http.Client
	BaseURL    string
	Timeout    time.Duration
}
