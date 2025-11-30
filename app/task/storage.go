package task

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var (
	manager *TaskManager
	once    sync.Once
	mu      sync.RWMutex
)

// GetTaskManager 获取任务管理器单例
func GetTaskManager() *TaskManager {
	once.Do(func() {
		manager = &TaskManager{
			tasks:    make(map[string]*Task),
			filePath: "tasks.json",
		}
		manager.loadTasks()
	})
	return manager
}

// loadTasks 从文件加载任务
func (tm *TaskManager) loadTasks() error {
	log.Printf("Loading tasks from file: %s\n", tm.filePath)
	mu.Lock()
	defer mu.Unlock()

	// 如果文件不存在，创建空文件
	if _, err := os.Stat(tm.filePath); os.IsNotExist(err) {
		log.Printf("Tasks file does not exist, creating empty file\n")
		tm.tasks = make(map[string]*Task)
		// 直接保存，不调用 saveTasks 避免重复加锁
		data, err := json.MarshalIndent([]*Task{}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal empty tasks: %v", err)
		}
		if err := ioutil.WriteFile(tm.filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write tasks file: %v", err)
		}
		log.Printf("Created empty tasks file\n")
		return nil
	}

	data, err := ioutil.ReadFile(tm.filePath)
	if err != nil {
		log.Printf("ERROR: Failed to read tasks file: %v\n", err)
		return fmt.Errorf("failed to read tasks file: %v", err)
	}

	if len(data) == 0 {
		log.Printf("Tasks file is empty\n")
		tm.tasks = make(map[string]*Task)
		return nil
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		log.Printf("ERROR: Failed to parse tasks file: %v\n", err)
		return fmt.Errorf("failed to parse tasks file: %v", err)
	}

	tm.tasks = make(map[string]*Task)
	for _, task := range tasks {
		tm.tasks[task.ID] = task
	}

	log.Printf("Loaded %d tasks from file\n", len(tm.tasks))
	return nil
}

// saveTasks 保存任务到文件
func (tm *TaskManager) saveTasks() error {
	log.Printf("saveTasks called, current tasks count: %d\n", len(tm.tasks))

	// 注意：saveTasks 不应该再次获取锁，因为调用者已经持有锁
	// 如果这里再次获取锁会导致死锁
	// mu.Lock()  // 注释掉，因为调用者已经持有锁
	// defer mu.Unlock()

	tasks := make([]*Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	log.Printf("Prepared %d tasks for saving\n", len(tasks))

	log.Printf("Marshaling tasks to JSON...\n")
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		log.Printf("ERROR: Failed to marshal tasks: %v\n", err)
		return fmt.Errorf("failed to marshal tasks: %v", err)
	}
	log.Printf("Tasks marshaled, data length: %d bytes\n", len(data))

	log.Printf("Writing to file: %s\n", tm.filePath)
	if err := ioutil.WriteFile(tm.filePath, data, 0644); err != nil {
		log.Printf("ERROR: Failed to write tasks file: %v\n", err)
		return fmt.Errorf("failed to write tasks file: %v", err)
	}
	log.Printf("Tasks saved successfully to file\n")

	return nil
}
