package task

import (
	"time"

	"gorm.io/gorm"
)

// Task 任务模型
type Task struct {
	ID            uint      `gorm:"primaryKey;autoIncrement" json:"id"`                  // 任务ID（自增主键）
	Title         string    `gorm:"type:varchar(255);not null" json:"title"`            // 任务标题（由AI推测）
	Content       string    `gorm:"type:text;not null" json:"content"`                   // 任务具体内容（用户输入）
	CreatorID     string    `gorm:"type:varchar(100);not null;index" json:"creator_id"` // 创建任务的用户ID
	CreateTime    time.Time `gorm:"type:datetime;not null;index" json:"create_time"`     // 布置时间
	DueTime       *time.Time `gorm:"type:datetime;null;index" json:"due_time"`          // 预计结束时间（可选）
	Status        string    `gorm:"type:varchar(20);not null;default:'pending';index" json:"status"` // 任务状态: pending, in_progress, completed, cancelled
	CompletedTime *time.Time `gorm:"type:datetime;null" json:"completed_time"`         // 完成时间（可选）
	
	// 关联关系
	Dependencies []TaskDependency `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"-"` // GORM关联，不序列化到JSON
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// TaskDependency 任务依赖关系模型
type TaskDependency struct {
	TaskID       uint `gorm:"primaryKey;index" json:"task_id"`
	DependencyID uint `gorm:"primaryKey;index" json:"dependency_id"`
}

// TableName 指定表名
func (TaskDependency) TableName() string {
	return "task_dependencies"
}

// TaskManager 任务管理器
type TaskManager struct {
	db *gorm.DB
}

// TaskStatus 任务状态常量
const (
	StatusPending     = "pending"
	StatusInProgress  = "in_progress"
	StatusCompleted   = "completed"
	StatusCancelled   = "cancelled"
)

// GetDependencyIDs 获取依赖任务ID列表（用于JSON序列化）
func (t *Task) GetDependencyIDs() []uint {
	ids := make([]uint, len(t.Dependencies))
	for i, dep := range t.Dependencies {
		ids[i] = dep.DependencyID
	}
	return ids
}

