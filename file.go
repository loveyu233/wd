package wd

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

// InitConfig 用来读取配置文件并填充提供的结构体。
func InitConfig(fp string, cfg any) error {
	if fp == "" || !IsPtr(cfg) {
		return errors.New("fp为空或cfg非指针")
	}

	file, err := os.ReadFile(fp)
	if err != nil {
		return err
	}

	switch filepath.Ext(fp) {
	case ".json":
		return json.Unmarshal(file, cfg)
	case ".yml", ".yaml":
		return yaml.Unmarshal(file, cfg)
	default:
		return errors.New("无效的文件扩展名")
	}
}

// GetFileContentType 用来根据文件前缀识别内容类型。
func GetFileContentType(file []byte) string {
	return http.DetectContentType(file)
}

// GetFileNameType 用来取得文件名的扩展名。
func GetFileNameType(fileName string) string {
	split := strings.Split(fileName, ".")
	if len(split) < 2 {
		return ""
	}
	return split[len(split)-1]
}

// ReadFileContent 用来读取指定路径的文件内容。
func ReadFileContent(filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

// GetCurrentLine 用来返回当前调用点的文件、函数与行号。
func GetCurrentLine() (folderName string, fileName string, funcName string, lineNumber int) {
	pc, file, line, _ := runtime.Caller(1)
	lineNumber = line
	funcName = runtime.FuncForPC(pc).Name()
	if idx := strings.LastIndex(funcName, "."); idx != -1 {
		funcName = funcName[idx+1:]
	}

	fileName = filepath.Base(file)
	folderName = filepath.Base(filepath.Dir(file))

	return
}

type UploadFileReq struct {
	fileKey           string
	fileValue         *multipart.FileHeader
	OtherParams       map[string]string
	Headers           map[string]string
	token             string
	url               string
	resp              any
	fileNameGen       func(fileName string) string
	FileName          string
	beforeRequestFunc func(req *UploadFileReq)
}

type UploadFileOption func(*UploadFileReq)

// WithUploadFileKey 自定义上传文件的表单键名。
func WithUploadFileKey(key string) UploadFileOption {
	return func(req *UploadFileReq) {
		req.fileKey = key
	}
}

// WithUploadBeforeRequestFunc 请求调用前，参数组装后的钩子函数
func WithUploadBeforeRequestFunc(after func(req *UploadFileReq)) UploadFileOption {
	return func(req *UploadFileReq) {
		req.beforeRequestFunc = after
	}
}

// WithUploadFileValue 指定需要上传的文件内容。
func WithUploadFileValue(file *multipart.FileHeader) UploadFileOption {
	return func(req *UploadFileReq) {
		req.fileValue = file
	}
}

// WithUploadFileFormData 设置额外的表单参数，后设置的值会覆盖同名键。
func WithUploadFileFormData(params map[string]string) UploadFileOption {
	return func(req *UploadFileReq) {
		if req.OtherParams == nil {
			req.OtherParams = map[string]string{}
		}
		for k, v := range params {
			req.OtherParams[k] = v
		}
	}
}

// WithUploadFileHeaders 添加自定义请求头。
func WithUploadFileHeaders(headers map[string]string) UploadFileOption {
	return func(req *UploadFileReq) {
		if req.Headers == nil {
			req.Headers = map[string]string{}
		}
		for k, v := range headers {
			req.Headers[k] = v
		}
	}
}

// WithUploadFileToken 添加token信息。
func WithUploadFileToken(token string) UploadFileOption {
	return func(req *UploadFileReq) {
		req.token = token
	}
}

// WithUploadFileURL 指定目标地址。
func WithUploadFileURL(url string) UploadFileOption {
	return func(req *UploadFileReq) {
		req.url = url
	}
}

// WithUploadFileResp 指定响应结果需要反序列化到的对象。
func WithUploadFileResp(resp any) UploadFileOption {
	return func(req *UploadFileReq) {
		req.resp = resp
	}
}

// WithUploadFileNameGenerator 提供自定义的文件名生成器。
func WithUploadFileNameGenerator(gen func(fileName string) string) UploadFileOption {
	return func(req *UploadFileReq) {
		req.fileNameGen = gen
	}
}

// UploadFileToTargetURL 上传文件到指定url的快捷操作
func UploadFileToTargetURL(options ...UploadFileOption) error {
	req := UploadFileReq{
		fileKey:     "file",
		OtherParams: map[string]string{},
	}
	for _, option := range options {
		if option != nil {
			option(&req)
		}
	}

	if req.resp == nil || !IsPtr(req.resp) {
		return errors.New("value必须为指针")
	}
	if req.fileValue == nil {
		return errors.New("file为空")
	}
	if req.OtherParams == nil {
		req.OtherParams = map[string]string{}
	}
	if req.fileNameGen == nil {
		req.fileNameGen = func(fileName string) string {
			fileType := GetFileNameType(req.fileValue.Filename)
			return fmt.Sprintf("%s.%s", GetUUID(), fileType)
		}
	}
	req.FileName = req.fileNameGen(req.fileValue.Filename)

	if req.beforeRequestFunc != nil {
		req.beforeRequestFunc(&req)
	}

	open, err := req.fileValue.Open()
	if err != nil {
		return err
	}
	request := R().
		SetFileReader(req.fileKey, req.FileName, open).
		SetFormData(req.OtherParams)

	if req.token != "" {
		request = request.SetAuthToken(strings.TrimSpace(strings.TrimPrefix(req.token, "Bearer")))
	}
	if len(req.Headers) > 0 {
		request = request.SetHeaders(req.Headers)
	}

	resp, err := request.
		Post(req.url)
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return errors.New("上传失败，响应状态码为：" + resp.Status())
	}

	return json.Unmarshal(resp.Body(), req.resp)
}
