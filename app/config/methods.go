package config

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

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
		ModelName := os.Getenv("ModelName")
		AutoPass := os.Getenv("AutoPass")
		MaxMsg := os.Getenv("MaxMsg")
		
		if ApiKey != "" {
			config.ApiKey = ApiKey
		}
		if ModelName != "" {
			config.ModelName = ModelName
		}
		if AutoPass == "true" {
			config.AutoPass = true
		}
		if MaxMsg != "" {
			config.MaxMsg = 15
		}
		
		// 设置默认值
		if config.ModelName == "" {
			config.ModelName = "deepseek-chat"
		}
	})
	return config
}

