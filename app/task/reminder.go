package task

import (
	"log"
	"time"
)

// StartReminderService 启动定时提醒服务
func StartReminderService(notifyFunc func(tasks []*Task)) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // 每小时检查一次
		defer ticker.Stop()
		
		for range ticker.C {
			tm := GetTaskManager()
			overdue := tm.GetOverdueTasks()
			
			if len(overdue) > 0 {
				log.Printf("Found %d overdue tasks\n", len(overdue))
				notifyFunc(overdue)
			}
			
			// 检查即将到期的任务（24小时内）
			upcoming := tm.GetUpcomingTasks(24 * time.Hour)
			if len(upcoming) > 0 {
				log.Printf("Found %d upcoming tasks\n", len(upcoming))
				notifyFunc(upcoming)
			}
		}
	}()
}

// GetUpcomingTasks 获取即将到期的任务
func (tm *TaskManager) GetUpcomingTasks(duration time.Duration) []*Task {
	now := time.Now()
	deadline := now.Add(duration)
	var tasks []*Task
	
	if err := tm.db.Preload("Dependencies").
		Where("status NOT IN ? AND due_time IS NOT NULL AND due_time > ? AND due_time < ?", 
			[]string{StatusCompleted, StatusCancelled}, now, deadline).
		Order("due_time ASC").
		Find(&tasks).Error; err != nil {
		log.Printf("ERROR: Failed to get upcoming tasks: %v\n", err)
		return []*Task{}
	}
	
	return tasks
}

