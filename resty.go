package wd

import (
	"encoding/json"
	"errors"
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

// RPost 快捷的post请求，参数headers为请求头，body为请求体，url为请求地址，value为返回值必须是指针，token按需传递
func RPost(headers map[string]string, body interface{}, url string, value any, token ...string) error {
	if !IsPtr(value) {
		return errors.New("value必须是一个指针")
	}
	request := R().SetHeaders(headers).SetBody(body)
	if len(token) > 0 {
		request = request.SetAuthToken(strings.TrimSpace(strings.TrimPrefix(token[0], "Bearer")))
	}
	resp, err := request.Post(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.New("请求失败，状态码为：" + resp.Status())
	}

	return json.Unmarshal(resp.Body(), value)
}

// RGet 快捷的get请求，参数headers为请求头，query为请求体，url为请求地址，value为返回值必须是指针，token按需传递
func RGet(headers map[string]string, query map[string]string, url string, value any, token ...string) error {
	if !IsPtr(value) {
		return errors.New("value必须是一个指针")
	}
	request := R().SetHeaders(headers).SetQueryParams(query)
	if len(token) > 0 {
		request = request.SetAuthToken(strings.TrimSpace(strings.TrimPrefix(token[0], "Bearer")))
	}
	resp, err := request.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode() != http.StatusOK {
		return errors.New("请求失败，状态码为：" + resp.Status())
	}

	return json.Unmarshal(resp.Body(), value)
}
