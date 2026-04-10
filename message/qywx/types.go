package qywx

// MediaType 表示企业微信机器人支持的媒体文件类型。
type MediaType string

const (
	// MediaTypeVoice 表示语音媒体。
	MediaTypeVoice MediaType = "voice"
	// MediaTypeFile 表示普通文件媒体。
	MediaTypeFile MediaType = "file"
)

// UploadResponse 表示企业微信上传媒体文件的结果。
type UploadResponse struct {
	Errcode   int    `json:"errcode"`
	Errmsg    string `json:"errmsg"`
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// SendResponse 表示企业微信机器人发送消息的结果。
type SendResponse struct {
	Errcode int    `json:"errcode"`
	Errmsg  string `json:"errmsg"`
}

// Message 表示企业微信机器人支持的消息体接口。
type Message interface {
	MessageType() string
}

// TextMessage 表示文本消息。
type TextMessage struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content             string   `json:"content"`
		MentionedList       []string `json:"mentioned_list,omitempty"`
		MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
	} `json:"text"`
}

// MarkdownMessage 表示 Markdown 消息。
type MarkdownMessage struct {
	Msgtype  string `json:"msgtype"`
	Markdown struct {
		Content string `json:"content"`
	} `json:"markdown"`
}

// ImageMessage 表示图片消息。
type ImageMessage struct {
	Msgtype string `json:"msgtype"`
	Image   struct {
		Base64 string `json:"base64,omitempty"`
		MD5    string `json:"md5,omitempty"`
	} `json:"image"`
}

// FileMessage 表示文件消息。
type FileMessage struct {
	Msgtype string `json:"msgtype"`
	File    struct {
		MediaID string `json:"media_id"`
	} `json:"file"`
}

// NewsMessage 表示图文消息。
type NewsMessage struct {
	Msgtype string `json:"msgtype"`
	News    struct {
		Articles []Article `json:"articles"`
	} `json:"news"`
}

// Article 表示图文消息中的文章项。
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	PicURL      string `json:"picurl"`
}
