package task

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// CreateTask 创建任务
func (tm *TaskManager) CreateTask(title, content, creator string, dueTime time.Time, dependencies []string) (*Task, error) {
	log.Printf("CreateTask called: title=%s, creator=%s, dependencies=%v\n", title, creator, dependencies)

	mu.Lock()
	log.Printf("CreateTask acquired write lock\n")
	defer func() {
		mu.Unlock()
		log.Printf("CreateTask released write lock\n")
	}()

	// 检查依赖是否存在且不形成循环
	log.Printf("Checking dependencies...\n")
	if err := tm.checkDependencies(dependencies, ""); err != nil {
		log.Printf("Dependency check failed: %v\n", err)
		return nil, err
	}
	log.Printf("Dependency check passed\n")

	log.Printf("Creating task object...\n")
	task := &Task{
		ID:           uuid.New().String(),
		Title:        title,
		Content:      content,
		Creator:      creator,
		CreateTime:   time.Now(),
		DueTime:      dueTime,
		Dependencies: dependencies,
		Status:       StatusPending,
	}
	log.Printf("Task object created: ID=%s\n", task.ID)

	log.Printf("Adding task to map...\n")
	tm.tasks[task.ID] = task
	log.Printf("Task added to map, current count: %d\n", len(tm.tasks))

	log.Printf("Saving tasks to file...\n")
	if err := tm.saveTasks(); err != nil {
		log.Printf("ERROR: Failed to save tasks: %v\n", err)
		delete(tm.tasks, task.ID)
		return nil, err
	}
	log.Printf("Tasks saved successfully\n")

	log.Printf("Created task: %s (ID: %s)\n", title, task.ID)
	return task, nil
}

// checkDependencies 检查依赖关系，防止循环依赖
func (tm *TaskManager) checkDependencies(dependencies []string, currentTaskID string) error {
	log.Printf("checkDependencies called: dependencies=%v, currentTaskID=%s\n", dependencies, currentTaskID)

	if len(dependencies) == 0 {
		log.Printf("No dependencies to check\n")
		return nil
	}

	visited := make(map[string]bool)

	var checkCycle func(taskID string) error
	checkCycle = func(taskID string) error {
		log.Printf("Checking dependency: %s\n", taskID)
		if taskID == currentTaskID {
			return fmt.Errorf("circular dependency detected")
		}

		if visited[taskID] {
			log.Printf("Dependency %s already visited, skipping\n", taskID)
			return nil // 已经检查过，避免重复
		}

		visited[taskID] = true

		task, exists := tm.tasks[taskID]
		if !exists {
			log.Printf("ERROR: Dependency task %s not found\n", taskID)
			return fmt.Errorf("dependency task %s not found", taskID)
		}

		log.Printf("Dependency task %s found, checking its dependencies...\n", taskID)
		// 递归检查依赖任务的依赖
		for _, depID := range task.Dependencies {
			if err := checkCycle(depID); err != nil {
				return err
			}
		}

		return nil
	}

	for _, depID := range dependencies {
		if err := checkCycle(depID); err != nil {
			log.Printf("Dependency check failed for %s: %v\n", depID, err)
			return err
		}
	}

	log.Printf("All dependencies checked successfully\n")
	return nil
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(id string) (*Task, bool) {
	mu.RLock()
	defer mu.RUnlock()

	task, exists := tm.tasks[id]
	return task, exists
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks(status string) []*Task {
	mu.RLock()
	defer mu.RUnlock()

	tasks := make([]*Task, 0)
	for _, task := range tm.tasks {
		if status == "" || task.Status == status {
			tasks = append(tasks, task)
		}
	}

	return tasks
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(id, status string) error {
	mu.Lock()
	defer mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return fmt.Errorf("task %s not found", id)
	}

	// 验证状态
	validStatuses := map[string]bool{
		StatusPending:    true,
		StatusInProgress: true,
		StatusCompleted:  true,
		StatusCancelled:  true,
	}
	if !validStatuses[status] {
		return fmt.Errorf("invalid status: %s", status)
	}

	task.Status = status
	if status == StatusCompleted {
		now := time.Now()
		task.CompletedTime = &now
	}

	if err := tm.saveTasks(); err != nil {
		return err
	}

	log.Printf("Updated task %s status to %s\n", id, status)
	return nil
}

// DeleteTask 删除任务
func (tm *TaskManager) DeleteTask(id string) error {
	mu.Lock()
	defer mu.Unlock()

	// 检查是否有其他任务依赖此任务
	for _, task := range tm.tasks {
		for _, depID := range task.Dependencies {
			if depID == id {
				return fmt.Errorf("cannot delete task %s: task %s depends on it", id, task.ID)
			}
		}
	}

	if _, exists := tm.tasks[id]; !exists {
		return fmt.Errorf("task %s not found", id)
	}

	delete(tm.tasks, id)

	if err := tm.saveTasks(); err != nil {
		return err
	}

	log.Printf("Deleted task %s\n", id)
	return nil
}

// GetTaskCount 获取任务数量
func (tm *TaskManager) GetTaskCount(status string) int {
	log.Printf("GetTaskCount called with status: '%s'\n", status)
	mu.RLock()
	log.Printf("GetTaskCount acquired read lock\n")
	defer func() {
		mu.RUnlock()
		log.Printf("GetTaskCount released read lock\n")
	}()

	if status == "" {
		count := len(tm.tasks)
		log.Printf("GetTaskCount returning total count: %d\n", count)
		return count
	}

	count := 0
	for _, task := range tm.tasks {
		if task.Status == status {
			count++
		}
	}
	log.Printf("GetTaskCount returning count for status '%s': %d\n", status, count)
	return count
}

// GetOverdueTasks 获取过期任务
func (tm *TaskManager) GetOverdueTasks() []*Task {
	mu.RLock()
	defer mu.RUnlock()

	now := time.Now()
	overdue := make([]*Task, 0)

	for _, task := range tm.tasks {
		if task.Status != StatusCompleted && task.Status != StatusCancelled {
			if task.DueTime.Before(now) {
				overdue = append(overdue, task)
			}
		}
	}

	return overdue
}

// FormatTaskForDisplay 格式化任务用于微信显示
func FormatTaskForDisplay(task *Task) string {
	statusText := map[string]string{
		StatusPending:    "待处理",
		StatusInProgress: "进行中",
		StatusCompleted:  "已完成",
		StatusCancelled:  "已取消",
	}

	status := statusText[task.Status]
	if status == "" {
		status = task.Status
	}

	result := fmt.Sprintf("📋 任务: %s\n", task.Title)
	result += fmt.Sprintf("状态: %s\n", status)
	result += fmt.Sprintf("创建人: %s\n", task.Creator)
	result += fmt.Sprintf("创建时间: %s\n", task.CreateTime.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("截止时间: %s\n", task.DueTime.Format("2006-01-02 15:04:05"))

	if task.Content != "" {
		result += fmt.Sprintf("内容: %s\n", task.Content)
	}

	if len(task.Dependencies) > 0 {
		result += fmt.Sprintf("依赖任务: %d个\n", len(task.Dependencies))
	}

	if task.CompletedTime != nil {
		result += fmt.Sprintf("完成时间: %s\n", task.CompletedTime.Format("2006-01-02 15:04:05"))
	}

	result += fmt.Sprintf("ID: %s", task.ID)

	return result
}

// FormatTaskListForDisplay 格式化任务列表用于微信显示
func FormatTaskListForDisplay(tasks []*Task) string {
	if len(tasks) == 0 {
		return "📋 暂无任务"
	}

	result := fmt.Sprintf("📋 任务列表 (共 %d 个):\n\n", len(tasks))

	for i, task := range tasks {
		statusEmoji := map[string]string{
			StatusPending:    "⏳",
			StatusInProgress: "🔄",
			StatusCompleted:  "✅",
			StatusCancelled:  "❌",
		}

		emoji := statusEmoji[task.Status]
		if emoji == "" {
			emoji = "📝"
		}

		result += fmt.Sprintf("%d. %s %s\n", i+1, emoji, task.Title)
		result += fmt.Sprintf("   创建人: %s | 截止: %s\n", task.Creator, task.DueTime.Format("2006-01-02 15:04"))

		if len(task.Dependencies) > 0 {
			result += fmt.Sprintf("   依赖: %d个任务\n", len(task.Dependencies))
		}

		result += "\n"
	}

	return result
}
