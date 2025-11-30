package task

import "time"

// Task 任务模型
type Task struct {
	ID            string    `json:"id"`              // 任务ID（唯一标识）
	Title         string    `json:"title"`           // 任务标题
	Content       string    `json:"content"`         // 任务具体内容
	Creator       string    `json:"creator"`         // 布置任务的用户
	CreateTime    time.Time `json:"create_time"`     // 布置时间
	DueTime       time.Time `json:"due_time"`        // 预计结束时间
	Dependencies  []string  `json:"dependencies"`    // 前置依赖任务ID列表
	Status        string    `json:"status"`          // 任务状态: pending, in_progress, completed, cancelled
	CompletedTime *time.Time `json:"completed_time"` // 完成时间（可选）
}

// TaskManager 任务管理器
type TaskManager struct {
	tasks    map[string]*Task
	filePath string
}

// TaskStatus 任务状态常量
const (
	StatusPending     = "pending"
	StatusInProgress  = "in_progress"
	StatusCompleted   = "completed"
	StatusCancelled   = "cancelled"
)

