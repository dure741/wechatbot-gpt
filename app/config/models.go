package config

// Configuration 项目配置
type Configuration struct {
	// DeepSeek API 密钥
	ApiKey string `json:"api_key"`
	// 模型名称 (如: deepseek-chat, deepseek-coder)
	ModelName string `json:"model_name"`
	// 自动通过好友
	AutoPass bool `json:"auto_pass"`
	MaxMsg   int  `json:"max_msg"`
}

