package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

// Configuration 项目配置
type Configuration struct {
	// gtp apikey
	ApiKey string `json:"api_key"`
	// 自动通过好友
	AutoPass bool `json:"auto_pass"`
	MaxMsg   int  `json:"max_msg"`
}

var config *Configuration
var once sync.Once

// LoadConfig 加载配置
func LoadConfig() *Configuration {
	once.Do(func() {
		// 从文件中读取
		config = &Configuration{}
		f, err := os.Open("config.json")
		if err != nil {
			log.Fatalf("open config err: %v", err)
			return
		}
		defer f.Close()
		encoder := json.NewDecoder(f)
		err = encoder.Decode(config)
		if err != nil {
			log.Fatalf("decode config err: %v", err)
			return
		}

		// 如果环境变量有配置，读取环境变量
		ApiKey := os.Getenv("ApiKey")
		AutoPass := os.Getenv("AutoPass")
		MaxMsg := os.Getenv("MaxMsg")
		if ApiKey != "" {
			config.ApiKey = ApiKey
		}
		if AutoPass == "true" {
			config.AutoPass = true
		}
		if MaxMsg != "" {
			config.MaxMsg = 15
		}
	})
	return config
}

const HelpText = `欢迎使用聊天机器人，你可以通过以下命令来使用聊天机器人：
1. 输入"help"查看帮助
2. 输入"role:" 定义角色行为 格式: "role:<你想定制的机器人角色行为>"
	-- 例如: role:你是一个非常有帮助的聊天机器人(这个是默认的角色行为)
3. 输入"get:role" 获取当前角色行为
4. 输入"get:session" 获取当前和你话题的聊天记录
5. 输入"换个话题" 重新开始一个话题(会清空当前和你话题的聊天记录保留角色行为)
---------------
更多功能正在开发中，敬请期待`
