package llm

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Provider AI 提供者接口
type Provider interface {
	// Chat 发送聊天请求
	Chat(messages []Message) (string, error)
	// GetModelName 获取模型名称
	GetModelName() string
	// GetBaseURL 获取 API 端点
	GetBaseURL() string
}

