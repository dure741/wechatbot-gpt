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
	// MySQL 数据库配置
	MySQL MySQLConfig `json:"mysql"`
}

// MySQLConfig MySQL数据库配置
type MySQLConfig struct {
	Host     string `json:"host"`     // 数据库主机地址
	Port     int    `json:"port"`     // 数据库端口
	User     string `json:"user"`     // 数据库用户名
	Password string `json:"password"` // 数据库密码
	Database string `json:"database"` // 数据库名称
	Charset  string `json:"charset"`  // 字符集，默认utf8mb4
}

