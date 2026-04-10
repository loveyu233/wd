package miniprogram

// StateType 表示小程序订阅消息发送时的目标环境。
type StateType string

const (
	// StateDeveloper 表示体验版。
	StateDeveloper StateType = "developer"
	// StateTrial 表示开发版。
	StateTrial StateType = "trial"
	// StateFormal 表示正式版。
	StateFormal StateType = "formal"
)

// SubscribeContent 表示发送订阅消息时的内容。
type SubscribeContent struct {
	ToUserOpenID string
	TemplateID   string
	Page         string
	State        StateType
	Data         map[string]map[string]any
}
