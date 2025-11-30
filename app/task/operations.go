package task

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// CreateTask 创建任务
func (tm *TaskManager) CreateTask(title, content, creatorID string, dueTime *time.Time, dependencies []uint) (*Task, error) {
	log.Printf("CreateTask called: title=%s, content_length=%d, creatorID=%s, dependencies=%v\n", title, len(content), creatorID, dependencies)

	// 验证必需参数
	if content == "" {
		return nil, fmt.Errorf("task content is required")
	}
	if creatorID == "" {
		return nil, fmt.Errorf("creator ID is required")
	}
	if title == "" {
		// 如果标题为空，使用内容的前50个字符作为标题
		if len(content) > 50 {
			title = content[:50] + "..."
		} else {
			title = content
		}
		log.Printf("Title was empty, using content preview: %s\n", title)
	}

	// 检查依赖是否存在且不形成循环
	if len(dependencies) > 0 {
		log.Printf("Checking dependencies...\n")
		if err := tm.checkDependencies(dependencies, 0); err != nil {
			log.Printf("Dependency check failed: %v\n", err)
			return nil, err
		}
		log.Printf("Dependency check passed\n")
	}

	// 创建任务对象
	log.Printf("Creating task object...\n")
	task := &Task{
		Title:         title,
		Content:       content,
		CreatorID:     creatorID,
		CreateTime:    time.Now(),
		DueTime:       dueTime,
		Status:        StatusPending,
		Dependencies:  make([]TaskDependency, 0),
	}

	// 使用事务保存任务和依赖关系
	err := tm.db.Transaction(func(tx *gorm.DB) error {
		// 保存任务
		if err := tx.Create(task).Error; err != nil {
			log.Printf("ERROR: Failed to create task in database: %v\n", err)
			return fmt.Errorf("failed to create task: %v", err)
		}
		log.Printf("Task created with ID: %d\n", task.ID)

		// 保存依赖关系
		if len(dependencies) > 0 {
			deps := make([]TaskDependency, len(dependencies))
			for i, depID := range dependencies {
				deps[i] = TaskDependency{
					TaskID:       task.ID,
					DependencyID: depID,
				}
			}
			if err := tx.Create(&deps).Error; err != nil {
				log.Printf("ERROR: Failed to create task dependencies: %v\n", err)
				return fmt.Errorf("failed to create task dependencies: %v", err)
			}
			task.Dependencies = deps
			log.Printf("Created %d dependencies\n", len(deps))
		}

		return nil
	})

	if err != nil {
		log.Printf("ERROR: Transaction failed: %v\n", err)
		return nil, err
	}

	log.Printf("Created task successfully: %s (ID: %d)\n", title, task.ID)
	return task, nil
}

// checkDependencies 检查依赖关系，防止循环依赖
func (tm *TaskManager) checkDependencies(dependencies []uint, currentTaskID uint) error {
	log.Printf("checkDependencies called: dependencies=%v, currentTaskID=%d\n", dependencies, currentTaskID)

	if len(dependencies) == 0 {
		log.Printf("No dependencies to check\n")
		return nil
	}

	visited := make(map[uint]bool)

	var checkCycle func(taskID uint) error
	checkCycle = func(taskID uint) error {
		log.Printf("Checking dependency: %d\n", taskID)
		if taskID == currentTaskID && currentTaskID != 0 {
			return fmt.Errorf("circular dependency detected")
		}

		if visited[taskID] {
			log.Printf("Dependency %d already visited, skipping\n", taskID)
			return nil // 已经检查过，避免重复
		}

		visited[taskID] = true

		// 检查任务是否存在
		var task Task
		if err := tm.db.First(&task, "id = ?", taskID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Printf("ERROR: Dependency task %d not found\n", taskID)
				return fmt.Errorf("dependency task %d not found", taskID)
			}
			return fmt.Errorf("failed to check dependency task %d: %v", taskID, err)
		}

		// 加载依赖关系
		var deps []TaskDependency
		if err := tm.db.Where("task_id = ?", taskID).Find(&deps).Error; err != nil {
			return fmt.Errorf("failed to load dependencies for task %d: %v", taskID, err)
		}

		log.Printf("Dependency task %d found, checking its dependencies...\n", taskID)
		// 递归检查依赖任务的依赖
		for _, dep := range deps {
			if err := checkCycle(dep.DependencyID); err != nil {
				return err
			}
		}

		return nil
	}

	for _, depID := range dependencies {
		if err := checkCycle(depID); err != nil {
			log.Printf("Dependency check failed for %d: %v\n", depID, err)
			return err
		}
	}

	log.Printf("All dependencies checked successfully\n")
	return nil
}

// GetTask 获取任务
func (tm *TaskManager) GetTask(id uint) (*Task, bool) {
	var task Task
	if err := tm.db.Preload("Dependencies").First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false
		}
		log.Printf("ERROR: Failed to get task: %v\n", err)
		return nil, false
	}
	return &task, true
}

// GetTaskByIDString 通过字符串ID获取任务（用于兼容）
func (tm *TaskManager) GetTaskByIDString(idStr string) (*Task, bool) {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, false
	}
	return tm.GetTask(uint(id))
}

// UpdateTaskDependencies 更新任务的依赖关系
func (tm *TaskManager) UpdateTaskDependencies(taskID uint, dependencies []uint) error {
	// 检查任务是否存在
	var task Task
	if err := tm.db.First(&task, "id = ?", taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task %d not found", taskID)
		}
		return fmt.Errorf("failed to get task: %v", err)
	}

	// 检查依赖是否存在且不形成循环
	if len(dependencies) > 0 {
		if err := tm.checkDependencies(dependencies, taskID); err != nil {
			return err
		}
	}

	// 使用事务更新依赖关系
	err := tm.db.Transaction(func(tx *gorm.DB) error {
		// 删除旧的依赖关系
		if err := tx.Where("task_id = ?", taskID).Delete(&TaskDependency{}).Error; err != nil {
			log.Printf("ERROR: Failed to delete old dependencies: %v\n", err)
			return fmt.Errorf("failed to delete old dependencies: %v", err)
		}

		// 创建新的依赖关系
		if len(dependencies) > 0 {
			deps := make([]TaskDependency, len(dependencies))
			for i, depID := range dependencies {
				deps[i] = TaskDependency{
					TaskID:       taskID,
					DependencyID: depID,
				}
			}
			if err := tx.Create(&deps).Error; err != nil {
				log.Printf("ERROR: Failed to create task dependencies: %v\n", err)
				return fmt.Errorf("failed to create task dependencies: %v", err)
			}
			log.Printf("Updated %d dependencies for task %d\n", len(deps), taskID)
		}

		return nil
	})

	return err
}

// ListTasks 列出所有任务
func (tm *TaskManager) ListTasks(status string) []*Task {
	var tasks []*Task
	query := tm.db.Preload("Dependencies")
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	if err := query.Order("create_time DESC").Find(&tasks).Error; err != nil {
		log.Printf("ERROR: Failed to list tasks: %v\n", err)
		return []*Task{}
	}

	return tasks
}

// UpdateTaskStatus 更新任务状态
func (tm *TaskManager) UpdateTaskStatus(id uint, status string) error {
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

	// 检查任务是否存在
	var task Task
	if err := tm.db.First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task %d not found", id)
		}
		return fmt.Errorf("failed to get task: %v", err)
	}

	// 更新状态
	updates := map[string]interface{}{
		"status": status,
	}
	if status == StatusCompleted {
		now := time.Now()
		updates["completed_time"] = &now
	}

	if err := tm.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update task status: %v", err)
	}

	log.Printf("Updated task %d status to %s\n", id, status)
	return nil
}

// UpdateTaskStatusByString 通过字符串ID更新任务状态（用于兼容）
func (tm *TaskManager) UpdateTaskStatusByString(idStr, status string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", idStr)
	}
	return tm.UpdateTaskStatus(uint(id), status)
}

// UpdateTask 更新任务的多个字段（标题、内容、截止时间等）
func (tm *TaskManager) UpdateTask(id uint, title *string, content *string, dueTime *time.Time) error {
	// 检查任务是否存在
	var task Task
	if err := tm.db.First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task %d not found", id)
		}
		return fmt.Errorf("failed to get task: %v", err)
	}

	// 构建更新字段
	updates := make(map[string]interface{})
	if title != nil {
		updates["title"] = *title
	}
	if content != nil {
		updates["content"] = *content
	}
	if dueTime != nil {
		updates["due_time"] = *dueTime
	} else if dueTime == nil && len(updates) > 0 {
		// 如果明确传入 nil，表示要清空截止时间
		// 这里不处理，因为 nil 指针无法区分"不更新"和"清空"
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// 更新任务
	if err := tm.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update task: %v", err)
	}

	log.Printf("Updated task %d: %v\n", id, updates)
	return nil
}

// DeleteTask 删除任务
func (tm *TaskManager) DeleteTask(id uint) error {
	// 检查是否有其他任务依赖此任务
	var count int64
	if err := tm.db.Model(&TaskDependency{}).Where("dependency_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check task dependencies: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("cannot delete task %d: %d task(s) depend on it", id, count)
	}

	// 检查任务是否存在
	var task Task
	if err := tm.db.First(&task, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("task %d not found", id)
		}
		return fmt.Errorf("failed to get task: %v", err)
	}

	// 删除任务（依赖关系会通过外键级联删除）
	if err := tm.db.Delete(&task).Error; err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	log.Printf("Deleted task %d\n", id)
	return nil
}

// DeleteTaskByString 通过字符串ID删除任务（用于兼容）
func (tm *TaskManager) DeleteTaskByString(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid task ID: %s", idStr)
	}
	return tm.DeleteTask(uint(id))
}

// GetTaskCount 获取任务数量
func (tm *TaskManager) GetTaskCount(status string) int {
	log.Printf("GetTaskCount called with status: '%s'\n", status)

	var count int64
	query := tm.db.Model(&Task{})
	
	if status != "" {
		query = query.Where("status = ?", status)
	}
	
	if err := query.Count(&count).Error; err != nil {
		log.Printf("ERROR: Failed to count tasks: %v\n", err)
		return 0
	}

	log.Printf("GetTaskCount returning count for status '%s': %d\n", status, count)
	return int(count)
}

// GetOverdueTasks 获取过期任务
func (tm *TaskManager) GetOverdueTasks() []*Task {
	now := time.Now()
	var tasks []*Task
	
	if err := tm.db.Preload("Dependencies").
		Where("status NOT IN ? AND due_time IS NOT NULL AND due_time < ?", []string{StatusCompleted, StatusCancelled}, now).
		Order("due_time ASC").
		Find(&tasks).Error; err != nil {
		log.Printf("ERROR: Failed to get overdue tasks: %v\n", err)
		return []*Task{}
	}

	return tasks
}

// FormatTaskForDisplay 格式化任务用于微信显示
func FormatTaskForDisplay(task *Task) string {
	return FormatTaskForDisplayWithManager(task, nil)
}

// FormatTaskForDisplayWithManager 格式化任务用于微信显示（带TaskManager用于获取依赖任务详情）
func FormatTaskForDisplayWithManager(task *Task, tm *TaskManager) string {
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
	result += fmt.Sprintf("创建人ID: %s\n", task.CreatorID)
	result += fmt.Sprintf("创建时间: %s\n", task.CreateTime.Format("2006-01-02 15:04:05"))
	
	if task.DueTime != nil {
		result += fmt.Sprintf("截止时间: %s\n", task.DueTime.Format("2006-01-02 15:04:05"))
	} else {
		result += fmt.Sprintf("截止时间: 未设置\n")
	}

	if task.Content != "" {
		result += fmt.Sprintf("内容: %s\n", task.Content)
	}

	dependencyIDs := task.GetDependencyIDs()
	if len(dependencyIDs) > 0 {
		result += fmt.Sprintf("依赖任务: ")
		for i, depID := range dependencyIDs {
			if i > 0 {
				result += ", "
			}
			// 如果提供了TaskManager，尝试获取依赖任务的标题
			if tm != nil {
				depTask, exists := tm.GetTask(depID)
				if exists {
					result += fmt.Sprintf("任务%d(%s)", depID, depTask.Title)
				} else {
					result += fmt.Sprintf("任务%d", depID)
				}
			} else {
				result += fmt.Sprintf("任务%d", depID)
			}
		}
		result += "\n"
	}

	if task.CompletedTime != nil {
		result += fmt.Sprintf("完成时间: %s\n", task.CompletedTime.Format("2006-01-02 15:04:05"))
	}

	result += fmt.Sprintf("ID: %d", task.ID)

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

		result += fmt.Sprintf("%d. %s %s (ID: %d)\n", i+1, emoji, task.Title, task.ID)
		result += fmt.Sprintf("   创建人ID: %s", task.CreatorID)
		
		if task.DueTime != nil {
			result += fmt.Sprintf(" | 截止: %s\n", task.DueTime.Format("2006-01-02 15:04"))
		} else {
			result += fmt.Sprintf(" | 截止: 未设置\n")
		}

		dependencyIDs := task.GetDependencyIDs()
		if len(dependencyIDs) > 0 {
			result += fmt.Sprintf("   依赖: %d个任务\n", len(dependencyIDs))
		}

		result += "\n"
	}

	return result
}
