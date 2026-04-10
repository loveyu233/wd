package qywx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const defaultBaseURL = "https://qyapi.weixin.qq.com/cgi-bin/webhook"

// Service 聚合企业微信机器人消息发送能力。
type Service struct {
	webhookKey string
	httpClient *http.Client
	baseURL    string
}

// New 用来根据配置初始化企业微信机器人服务。
func New(config Config) (*Service, error) {
	if config.WebhookKey == "" {
		return nil, errors.New("企业微信机器人 webhook key 不能为空")
	}
	client := config.HTTPClient
	if client == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Service{webhookKey: config.WebhookKey, httpClient: client, baseURL: baseURL}, nil
}

// SetTimeout 用来修改 HTTP 客户端超时时间。
func (s *Service) SetTimeout(timeout time.Duration) {
	s.httpClient.Timeout = timeout
}

// SetHTTPClient 用来替换底层 HTTP 客户端。
func (s *Service) SetHTTPClient(client *http.Client) error {
	if client == nil {
		return errors.New("HTTP 客户端不能为空")
	}
	s.httpClient = client
	return nil
}

// UploadMedia 用来上传媒体文件。
func (s *Service) UploadMedia(reader io.Reader, filename string, mediaType MediaType) (*UploadResponse, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取数据失败: %w", err)
	}
	contentType := http.DetectContentType(data)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", filename)
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %w", err)
	}
	if _, err = part.Write(data); err != nil {
		return nil, fmt.Errorf("写入文件内容失败: %w", err)
	}
	_ = writer.WriteField("filename", filename)
	_ = writer.WriteField("filelength", fmt.Sprintf("%d", len(data)))
	_ = writer.WriteField("content-type", contentType)
	if err = writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭 multipart writer 失败: %w", err)
	}
	url := fmt.Sprintf("%s/upload_media?key=%s&type=%s", s.baseURL, s.webhookKey, mediaType)
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("上传失败, 状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}
	var uploadResp UploadResponse
	if err = json.Unmarshal(respBody, &uploadResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if uploadResp.Errcode != 0 {
		return nil, fmt.Errorf("上传失败: %s", uploadResp.Errmsg)
	}
	return &uploadResp, nil
}

// UploadMediaFromFile 用来从文件路径上传媒体文件。
func (s *Service) UploadMediaFromFile(mediaType MediaType, filePath string) (*UploadResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()
	return s.UploadMedia(file, filepath.Base(filePath), mediaType)
}

func (TextMessage) MessageType() string     { return "text" }
func (MarkdownMessage) MessageType() string { return "markdown" }
func (ImageMessage) MessageType() string    { return "image" }
func (FileMessage) MessageType() string     { return "file" }
func (NewsMessage) MessageType() string     { return "news" }

// NewTextMessage 用来构造文本消息。
func NewTextMessage(content string) *TextMessage {
	msg := &TextMessage{Msgtype: "text"}
	msg.Text.Content = content
	return msg
}

// AddMention 用来给文本消息追加 @用户。
func (t *TextMessage) AddMention(userIDs ...string) *TextMessage {
	t.Text.MentionedList = append(t.Text.MentionedList, userIDs...)
	return t
}

// AddMentionMobile 用来给文本消息追加 @手机号。
func (t *TextMessage) AddMentionMobile(mobiles ...string) *TextMessage {
	t.Text.MentionedMobileList = append(t.Text.MentionedMobileList, mobiles...)
	return t
}

// NewMarkdownMessage 用来构造 Markdown 消息。
func NewMarkdownMessage(content string) *MarkdownMessage {
	msg := &MarkdownMessage{Msgtype: "markdown"}
	msg.Markdown.Content = content
	return msg
}

// NewFileMessage 用来构造文件消息。
func NewFileMessage(mediaID string) *FileMessage {
	msg := &FileMessage{Msgtype: "file"}
	msg.File.MediaID = mediaID
	return msg
}

// NewNewsMessage 用来构造图文消息。
func NewNewsMessage(articles ...Article) *NewsMessage {
	msg := &NewsMessage{Msgtype: "news"}
	msg.News.Articles = articles
	return msg
}

// AddArticle 用来给图文消息追加文章。
func (n *NewsMessage) AddArticle(title, description, url, picURL string) *NewsMessage {
	n.News.Articles = append(n.News.Articles, Article{Title: title, Description: description, URL: url, PicURL: picURL})
	return n
}

// SendMessage 用来发送机器人消息。
func (s *Service) SendMessage(msg Message) (*SendResponse, error) {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("序列化消息失败: %w", err)
	}
	url := fmt.Sprintf("%s/send?key=%s", s.baseURL, s.webhookKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("发送失败, 状态码: %d, 响应: %s", resp.StatusCode, string(respBody))
	}
	var sendResp SendResponse
	if err = json.Unmarshal(respBody, &sendResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if sendResp.Errcode != 0 {
		return nil, fmt.Errorf("发送失败: %s", sendResp.Errmsg)
	}
	return &sendResp, nil
}

// SendText 用来发送文本消息。
func (s *Service) SendText(content string) (*SendResponse, error) {
	return s.SendMessage(NewTextMessage(content))
}

// SendMarkdown 用来发送 Markdown 消息。
func (s *Service) SendMarkdown(content string) (*SendResponse, error) {
	return s.SendMessage(NewMarkdownMessage(content))
}

// SendFile 用来从文件路径发送文件消息。
func (s *Service) SendFile(filePath string, mediaType MediaType) (*SendResponse, error) {
	uploadResp, err := s.UploadMediaFromFile(mediaType, filePath)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}
	return s.SendMessage(NewFileMessage(uploadResp.MediaID))
}

// SendFileFromReader 用来从 io.Reader 发送文件消息。
func (s *Service) SendFileFromReader(mediaType MediaType, reader io.Reader, filename string) (*SendResponse, error) {
	uploadResp, err := s.UploadMedia(reader, filename, mediaType)
	if err != nil {
		return nil, fmt.Errorf("上传文件失败: %w", err)
	}
	return s.SendMessage(NewFileMessage(uploadResp.MediaID))
}

// SendNews 用来发送图文消息。
func (s *Service) SendNews(articles ...Article) (*SendResponse, error) {
	return s.SendMessage(NewNewsMessage(articles...))
}
