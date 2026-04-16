package wd

import (
	"errors"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

const smsDefaultEndpoint = "dysmsapi.aliyuncs.com"

// SMSConfig 用来描述阿里云短信服务的配置。
type SMSConfig struct {
	CredentialConfig *credential.Config
	Endpoint         string
}

// SMSService 聚合阿里云短信发送能力。
type SMSService struct {
	client *dysmsapi20170525.Client
}

// NewSMS 用来根据凭证配置初始化短信服务。
func NewSMS(config SMSConfig) (*SMSService, error) {
	if config.CredentialConfig == nil {
		return nil, errors.New("短信凭证配置不能为空")
	}
	newCredential, err := credential.NewCredential(config.CredentialConfig)
	if err != nil {
		return nil, err
	}
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = smsDefaultEndpoint
	}
	client, err := dysmsapi20170525.NewClient(&openapi.Config{Credential: newCredential, Endpoint: tea.String(endpoint)})
	if err != nil {
		return nil, err
	}
	return &SMSService{client: client}, nil
}

// NewSMSWithAccessKey 用来通过 access key 初始化短信服务。
func NewSMSWithAccessKey(accessKeyID, accessKeySecret string) (*SMSService, error) {
	return NewSMS(SMSConfig{CredentialConfig: new(credential.Config).SetType("access_key").SetAccessKeyId(accessKeyID).SetAccessKeySecret(accessKeySecret)})
}

// NewSMSWithClient 用来复用外部传入的短信客户端。
func NewSMSWithClient(client *dysmsapi20170525.Client) (*SMSService, error) {
	if client == nil {
		return nil, errors.New("短信客户端不能为空")
	}
	return &SMSService{client: client}, nil
}

// Client 用来返回底层短信客户端。
func (s *SMSService) Client() *dysmsapi20170525.Client {
	return s.client
}

// SendMsg 用来发送单条短信。
func (s *SMSService) SendMsg(req *dysmsapi20170525.SendSmsRequest) (err error) {
	if s.client == nil {
		return errors.New("短信客户端未初始化")
	}
	runtime := &util.RuntimeOptions{}
	defer func() {
		if r := tea.Recover(recover()); r != nil {
			err = r
		}
	}()
	_, err = s.client.SendSmsWithOptions(req, runtime)
	return err
}

// SendSimpleMsg 用来发送简化版单条短信。
func (s *SMSService) SendSimpleMsg(targetPhoneNumber, signName, templateCode, templateParam string) error {
	return s.SendMsg(&dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(targetPhoneNumber),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(templateParam),
	})
}

// SendBatchSms 用来批量发送短信。
func (s *SMSService) SendBatchSms(req *dysmsapi20170525.SendBatchSmsRequest) (err error) {
	if s.client == nil {
		return errors.New("短信客户端未初始化")
	}
	runtime := &util.RuntimeOptions{}
	defer func() {
		if r := tea.Recover(recover()); r != nil {
			err = r
		}
	}()
	_, err = s.client.SendBatchSmsWithOptions(req, runtime)
	return err
}

// SendSimpleBatchMsg 用来发送简化版批量短信。
func (s *SMSService) SendSimpleBatchMsg(targetPhoneNumbers, signNames, templateCode, templateParams string) error {
	return s.SendBatchSms(&dysmsapi20170525.SendBatchSmsRequest{
		PhoneNumberJson:   tea.String(targetPhoneNumbers),
		SignNameJson:      tea.String(signNames),
		TemplateCode:      tea.String(templateCode),
		TemplateParamJson: tea.String(templateParams),
	})
}
