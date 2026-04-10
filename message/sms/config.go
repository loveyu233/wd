package sms

import credential "github.com/aliyun/credentials-go/credentials"

// Config 用来描述阿里云短信服务的配置。
type Config struct {
	CredentialConfig *credential.Config
	Endpoint         string
}
