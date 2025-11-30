package task

import (
	"log"
	"sync"
)

var (
	manager *TaskManager
	once    sync.Once
)

// GetTaskManager 获取任务管理器单例
func GetTaskManager() *TaskManager {
	once.Do(func() {
		// 确保数据库已初始化
		if err := InitDatabase(); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		manager = &TaskManager{
			db: GetDB(),
		}
		log.Printf("TaskManager initialized with GORM\n")
	})
	return manager
}
