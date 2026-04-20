package wd

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	defaultRestyClient *resty.Client
	restyOnce          sync.Once
)

// initRestyClient 用来按默认配置初始化 Resty 客户端。
func initRestyClient() {
	defaultRestyClient = resty.New()
	defaultRestyClient.
		SetTimeout(5 * time.Second).
		SetRetryCount(0)
}

// RestyClient 用来返回带懒加载的 Resty 单例。
func RestyClient() *resty.Client {
	restyOnce.Do(initRestyClient)
	return defaultRestyClient
}

// R 用来基于默认客户端创建请求。
func R() *resty.Request {
	return RestyClient().R()
}

type restySendFunc func(*resty.Request) (*resty.Response, error)

func validatePointerTarget(value any) error {
	if !IsPtr(value) {
		return errors.New("value必须是一个指针")
	}
	return nil
}

func decodeJSONResponse(prefix string, resp *resty.Response, value any) error {
	if resp.StatusCode() != http.StatusOK {
		return newResponseStatusError(prefix, resp)
	}
	return json.Unmarshal(resp.Body(), value)
}

func executeJSONRequest(
	prefix string,
	value any,
	token string,
	configure func(*resty.Request) *resty.Request,
	send restySendFunc,
) error {
	if err := validatePointerTarget(value); err != nil {
		return err
	}

	request := R()
	if configure != nil {
		request = configure(request)
	}
	if token != "" {
		request = setRequestAuthToken(request, token)
	}

	resp, err := send(request)
	if err != nil {
		return err
	}
	return decodeJSONResponse(prefix, resp, value)
}

// RPost 快捷的post请求，参数headers为请求头，body为请求体，url为请求地址，value为返回值必须是指针，token按需传递
func RPost(headers map[string]string, body interface{}, url string, value any, token ...string) error {
	var authToken string
	if len(token) > 0 {
		authToken = token[0]
	}
	return executeJSONRequest(
		"请求失败",
		value,
		authToken,
		func(request *resty.Request) *resty.Request {
			return request.SetHeaders(headers).SetBody(body)
		},
		func(request *resty.Request) (*resty.Response, error) {
			return request.Post(url)
		},
	)
}

// RGet 快捷的get请求，参数headers为请求头，query为请求体，url为请求地址，value为返回值必须是指针，token按需传递
func RGet(headers map[string]string, query map[string]string, url string, value any, token ...string) error {
	var authToken string
	if len(token) > 0 {
		authToken = token[0]
	}
	return executeJSONRequest(
		"请求失败",
		value,
		authToken,
		func(request *resty.Request) *resty.Request {
			return request.SetHeaders(headers).SetQueryParams(query)
		},
		func(request *resty.Request) (*resty.Response, error) {
			return request.Get(url)
		},
	)
}

func setRequestAuthToken(request *resty.Request, token string) *resty.Request {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer"))
	if token == "" {
		return request
	}
	return request.SetAuthToken(token)
}

func newResponseStatusError(prefix string, resp *resty.Response) error {
	return fmt.Errorf("%s，状态码为：%s", prefix, resp.Status())
}
