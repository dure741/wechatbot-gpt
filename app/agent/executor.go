package agent

import (
	"fmt"
	"log"
	"time"

	"github.com/869413421/wechatbot/app/task"
)

// Executor Agent 执行器
type Executor struct {
}

// NewExecutor 创建 Agent 执行器
func NewExecutor() *Executor {
	return &Executor{}
}

// ExecuteCommand 执行命令
func (e *Executor) ExecuteCommand(command string, args map[string]interface{}) (string, error) {
	log.Printf("Agent executing command: %s with args: %v\n", command, args)

	switch command {
	case "create_task":
		return e.createTask(args)
	case "list_tasks":
		return e.listTasks(args)
	case "get_task_count":
		return e.getTaskCount(args)
	case "update_task_status":
		return e.updateTaskStatus(args)
	case "get_task":
		return e.getTask(args)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// createTask 创建任务
func (e *Executor) createTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	// 解析参数
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	creator, _ := args["creator"].(string)

	if title == "" {
		return "", fmt.Errorf("task title is required")
	}
	if creator == "" {
		return "", fmt.Errorf("task creator is required")
	}

	// 解析截止时间
	var dueTime time.Time
	if dueTimeStr, ok := args["due_time"].(string); ok && dueTimeStr != "" {
		var err error
		dueTime, err = time.Parse("2006-01-02 15:04:05", dueTimeStr)
		if err != nil {
			// 尝试其他格式
			dueTime, err = time.Parse("2006-01-02", dueTimeStr)
			if err != nil {
				// 默认24小时后
				dueTime = time.Now().Add(24 * time.Hour)
			}
		}
	} else {
		// 默认24小时后
		dueTime = time.Now().Add(24 * time.Hour)
	}

	// 解析依赖任务
	var dependencies []string
	if deps, ok := args["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				dependencies = append(dependencies, depStr)
			}
		}
	}

	// 创建任务
	log.Printf("Calling CreateTask...\n")
	createdTask, err := tm.CreateTask(title, content, creator, dueTime, dependencies)
	if err != nil {
		log.Printf("ERROR: CreateTask failed: %v\n", err)
		return "", fmt.Errorf("failed to create task: %v", err)
	}
	log.Printf("CreateTask succeeded, task ID: %s\n", createdTask.ID)

	log.Printf("Formatting task for display...\n")
	result := fmt.Sprintf("任务创建成功！\n%s", task.FormatTaskForDisplay(createdTask))
	log.Printf("createTask returning result, length: %d\n", len(result))
	return result, nil
}

// listTasks 列出任务
func (e *Executor) listTasks(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	status, _ := args["status"].(string)
	tasks := tm.ListTasks(status)

	if len(tasks) == 0 {
		return "📋 暂无任务", nil
	}

	return task.FormatTaskListForDisplay(tasks), nil
}

// getTaskCount 获取任务数量
func (e *Executor) getTaskCount(args map[string]interface{}) (string, error) {
	log.Printf("getTaskCount called with args: %v\n", args)

	tm := task.GetTaskManager()
	log.Printf("TaskManager obtained\n")

	status, _ := args["status"].(string)
	log.Printf("Getting task count for status: '%s'\n", status)

	count := tm.GetTaskCount(status)
	log.Printf("Task count retrieved: %d\n", count)

	statusText := map[string]string{
		"":                    "全部",
		task.StatusPending:    "待处理",
		task.StatusInProgress: "进行中",
		task.StatusCompleted:  "已完成",
		task.StatusCancelled:  "已取消",
	}

	text := statusText[status]
	if text == "" {
		text = status
	}

	result := fmt.Sprintf("📊 %s任务数量: %d 个", text, count)
	log.Printf("getTaskCount returning: %s\n", result)
	return result, nil
}

// updateTaskStatus 更新任务状态
func (e *Executor) updateTaskStatus(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskID, _ := args["task_id"].(string)
	status, _ := args["status"].(string)

	if taskID == "" {
		return "", fmt.Errorf("task_id is required")
	}
	if status == "" {
		return "", fmt.Errorf("status is required")
	}

	err := tm.UpdateTaskStatus(taskID, status)
	if err != nil {
		return "", fmt.Errorf("failed to update task status: %v", err)
	}

	return fmt.Sprintf("任务状态已更新为: %s", status), nil
}

// getTask 获取单个任务
func (e *Executor) getTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskID, _ := args["task_id"].(string)
	if taskID == "" {
		return "", fmt.Errorf("task_id is required")
	}

	t, exists := tm.GetTask(taskID)
	if !exists {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	return task.FormatTaskForDisplay(t), nil
}

// GetAvailableCommands 获取可用命令列表
func (e *Executor) GetAvailableCommands() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "create_task",
			"description": "创建新任务。当用户说'记录任务'时使用此功能。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":        "string",
						"description": "任务标题（必需）",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "任务具体内容",
					},
					"creator": map[string]interface{}{
						"type":        "string",
						"description": "布置任务的用户名称（必需）",
					},
					"due_time": map[string]interface{}{
						"type":        "string",
						"description": "预计结束时间，格式：2006-01-02 15:04:05 或 2006-01-02，默认为24小时后",
					},
					"dependencies": map[string]interface{}{
						"type":        "array",
						"description": "前置依赖任务ID列表",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"title", "creator"},
			},
		},
		{
			"name":        "list_tasks",
			"description": "列出所有任务或指定状态的任务",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "任务状态筛选：pending（待处理）、in_progress（进行中）、completed（已完成）、cancelled（已取消），为空则列出所有任务",
					},
				},
			},
		},
		{
			"name":        "get_task_count",
			"description": "获取任务数量统计",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "任务状态筛选，为空则统计全部任务",
					},
				},
			},
		},
		{
			"name":        "get_task",
			"description": "根据任务ID获取任务详情",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "任务ID（必需）",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "update_task_status",
			"description": "更新任务状态",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "任务ID（必需）",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "新状态：pending（待处理）、in_progress（进行中）、completed（已完成）、cancelled（已取消）",
					},
				},
				"required": []string{"task_id", "status"},
			},
		},
	}
}
