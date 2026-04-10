package sms

import (
	"errors"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	credential "github.com/aliyun/credentials-go/credentials"
)

const defaultEndpoint = "dysmsapi.aliyuncs.com"

// Service 聚合阿里云短信发送能力。
type Service struct {
	client *dysmsapi20170525.Client
}

// New 用来根据凭证配置初始化短信服务。
func New(config Config) (*Service, error) {
	if config.CredentialConfig == nil {
		return nil, errors.New("短信凭证配置不能为空")
	}
	newCredential, err := credential.NewCredential(config.CredentialConfig)
	if err != nil {
		return nil, err
	}
	endpoint := config.Endpoint
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	client, err := dysmsapi20170525.NewClient(&openapi.Config{Credential: newCredential, Endpoint: tea.String(endpoint)})
	if err != nil {
		return nil, err
	}
	return &Service{client: client}, nil
}

// NewWithAccessKey 用来通过 access key 初始化短信服务。
func NewWithAccessKey(accessKeyID, accessKeySecret string) (*Service, error) {
	return New(Config{CredentialConfig: new(credential.Config).SetType("access_key").SetAccessKeyId(accessKeyID).SetAccessKeySecret(accessKeySecret)})
}

// NewWithClient 用来复用外部传入的短信客户端。
func NewWithClient(client *dysmsapi20170525.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("短信客户端不能为空")
	}
	return &Service{client: client}, nil
}

// SendMsg 用来发送单条短信。
func (s *Service) SendMsg(req *dysmsapi20170525.SendSmsRequest) error {
	if s.client == nil {
		return errors.New("短信客户端未初始化")
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				e = r
			}
		}()
		_, err := s.client.SendSmsWithOptions(req, runtime)
		return err
	}()
	return tryErr
}

// SendSimpleMsg 用来发送简化版单条短信。
func (s *Service) SendSimpleMsg(targetPhoneNumber, signName, templateCode, templateParam string) error {
	return s.SendMsg(&dysmsapi20170525.SendSmsRequest{
		PhoneNumbers:  tea.String(targetPhoneNumber),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(templateParam),
	})
}

// SendBatchSms 用来批量发送短信。
func (s *Service) SendBatchSms(req *dysmsapi20170525.SendBatchSmsRequest) error {
	if s.client == nil {
		return errors.New("短信客户端未初始化")
	}
	runtime := &util.RuntimeOptions{}
	tryErr := func() (e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				e = r
			}
		}()
		_, err := s.client.SendBatchSmsWithOptions(req, runtime)
		return err
	}()
	return tryErr
}

// SendSimpleBatchMsg 用来发送简化版批量短信。
func (s *Service) SendSimpleBatchMsg(targetPhoneNumbers, signNames, templateCode, templateParams string) error {
	return s.SendBatchSms(&dysmsapi20170525.SendBatchSmsRequest{
		PhoneNumberJson:   tea.String(targetPhoneNumbers),
		SignNameJson:      tea.String(signNames),
		TemplateCode:      tea.String(templateCode),
		TemplateParamJson: tea.String(templateParams),
	})
}
