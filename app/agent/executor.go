package agent

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
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
	case "update_task":
		return e.updateTask(args)
	case "get_task":
		return e.getTask(args)
	case "delete_task":
		return e.deleteTask(args)
	case "search_tasks":
		return e.searchTasks(args)
	case "get_overdue_tasks":
		return e.getOverdueTasks(args)
	case "get_upcoming_tasks":
		return e.getUpcomingTasks(args)
	case "update_task_dependencies":
		return e.updateTaskDependencies(args)
	default:
		return "", fmt.Errorf("unknown command: %s", command)
	}
}

// createTask 创建任务
func (e *Executor) createTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	// 解析必需参数：content（任务内容）和creator_id（用户ID）
	content, _ := args["content"].(string)
	creatorID, _ := args["creator_id"].(string)

	// 验证必需参数
	if content == "" {
		return "", fmt.Errorf("任务内容不能为空")
	}
	if creatorID == "" {
		return "", fmt.Errorf("创建人ID不能为空")
	}

	// 解析可选参数：title（由AI推测，如果为空则使用内容预览）
	title, _ := args["title"].(string)

	// 解析截止时间（可选）
	var dueTime *time.Time
	if dueTimeStr, ok := args["due_time"].(string); ok && dueTimeStr != "" {
		parsedTime, err := e.parseDueTime(dueTimeStr)
		if err != nil {
			log.Printf("WARNING: Failed to parse due_time '%s': %v, will set to nil\n", dueTimeStr, err)
			dueTime = nil
		} else {
			dueTime = &parsedTime
		}
	}

	// 解析依赖任务（可选）
	var dependencies []uint
	if deps, ok := args["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			var depID uint
			switch v := dep.(type) {
			case string:
				id, err := strconv.ParseUint(v, 10, 32)
				if err != nil {
					log.Printf("WARNING: Invalid dependency ID '%s': %v, skipping\n", v, err)
					continue
				}
				depID = uint(id)
			case float64:
				depID = uint(v)
			case int:
				depID = uint(v)
			default:
				log.Printf("WARNING: Invalid dependency type: %T, skipping\n", v)
				continue
			}
			dependencies = append(dependencies, depID)
		}
	}

	// 创建任务
	log.Printf("Creating task: title='%s', content_length=%d, creatorID=%s, dueTime=%v, dependencies=%v\n", 
		title, len(content), creatorID, dueTime, dependencies)
	if dueTimeStr, ok := args["due_time"].(string); ok {
		log.Printf("Raw due_time from AI: '%s'\n", dueTimeStr)
	}
	
	createdTask, err := tm.CreateTask(title, content, creatorID, dueTime, dependencies)
	if err != nil {
		log.Printf("ERROR: CreateTask failed: %v\n", err)
		return "", fmt.Errorf("创建任务失败: %v", err)
	}
	
	log.Printf("CreateTask succeeded, task ID: %d\n", createdTask.ID)

	result := fmt.Sprintf("✅ 任务创建成功！\n%s", task.FormatTaskForDisplayWithManager(createdTask, tm))
	return result, nil
}

// parseDueTime 解析截止时间，支持多种格式和自然语言
// AI应该已经将自然语言转换为标准格式，这里主要处理标准格式，但也支持一些自然语言作为备用
func (e *Executor) parseDueTime(timeStr string) (time.Time, error) {
	now := time.Now()
	timeStr = strings.TrimSpace(timeStr)
	timeStrLower := strings.ToLower(timeStr)
	
	// 处理自然语言（AI应该已经转换，但这里作为备用）
	if strings.Contains(timeStrLower, "今天") {
		// 提取时间部分
		timePart := extractTimeFromString(timeStr)
		if timePart != "" {
			return parseTimeForDate(now, timePart)
		}
		// 如果没有时间，默认今天23:59:59
		return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.Local), nil
	}
	
	if strings.Contains(timeStrLower, "明天") {
		tomorrow := now.AddDate(0, 0, 1)
		timePart := extractTimeFromString(timeStr)
		if timePart != "" {
			return parseTimeForDate(tomorrow, timePart)
		}
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 23, 59, 59, 0, time.Local), nil
	}
	
	if strings.Contains(timeStrLower, "后天") {
		dayAfterTomorrow := now.AddDate(0, 0, 2)
		timePart := extractTimeFromString(timeStr)
		if timePart != "" {
			return parseTimeForDate(dayAfterTomorrow, timePart)
		}
		return time.Date(dayAfterTomorrow.Year(), dayAfterTomorrow.Month(), dayAfterTomorrow.Day(), 23, 59, 59, 0, time.Local), nil
	}
	
	// 尝试多种标准时间格式
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间格式: %s", timeStr)
}

// extractTimeFromString 从字符串中提取时间部分（如 "12:00", "15:30", "13点", "下午4点"）
func extractTimeFromString(s string) string {
	// 先尝试匹配 HH:MM 或 HH:MM:SS 格式
	re := regexp.MustCompile(`(\d{1,2}):(\d{2})(?::(\d{2}))?`)
	matches := re.FindStringSubmatch(s)
	if len(matches) > 0 {
		return matches[0]
	}
	
	// 匹配中文格式：HH点MM分 或 HH点
	reCN := regexp.MustCompile(`(\d{1,2})(?:点|时)(?:(\d{2})(?:分)?)?`)
	matchesCN := reCN.FindStringSubmatch(s)
	if len(matchesCN) >= 2 {
		hour := matchesCN[1]
		minute := "00"
		if len(matchesCN) >= 3 && matchesCN[2] != "" {
			minute = matchesCN[2]
		}
		return hour + ":" + minute
	}
	
	// 匹配"下午X点"、"上午X点"等
	rePM := regexp.MustCompile(`(?:下午|晚上)(\d{1,2})(?:点|时)`)
	matchesPM := rePM.FindStringSubmatch(s)
	if len(matchesPM) >= 2 {
		hour, _ := strconv.Atoi(matchesPM[1])
		if hour < 12 {
			hour += 12 // 下午转换为24小时制
		}
		return fmt.Sprintf("%d:00", hour)
	}
	
	reAM := regexp.MustCompile(`(?:上午|早上)(\d{1,2})(?:点|时)`)
	matchesAM := reAM.FindStringSubmatch(s)
	if len(matchesAM) >= 2 {
		return matchesAM[1] + ":00"
	}
	
	return ""
}

// parseTimeForDate 为指定日期解析时间字符串
func parseTimeForDate(date time.Time, timeStr string) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}
	
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, fmt.Errorf("invalid hour: %s", parts[0])
	}
	
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return time.Time{}, fmt.Errorf("invalid minute: %s", parts[1])
	}
	
	second := 0
	if len(parts) > 2 {
		second, _ = strconv.Atoi(parts[2])
	}
	
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, time.Local), nil
}

// listTasks 列出任务（支持查看所有任务或按用户筛选）
func (e *Executor) listTasks(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	status, _ := args["status"].(string)
	creatorID, _ := args["creator_id"].(string)
	
	// 如果传入了creator_id，使用它；否则查看所有任务
	tasks := tm.ListTasks(status, creatorID)

	if len(tasks) == 0 {
		if creatorID != "" {
			return "📋 该用户暂无任务", nil
		}
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

	taskIDRaw := args["task_id"]
	status, _ := args["status"].(string)

	if status == "" {
		return "", fmt.Errorf("status is required")
	}

	// 解析任务ID
	var taskID uint
	switch v := taskIDRaw.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid task_id: %s", v)
		}
		taskID = uint(id)
	case float64:
		taskID = uint(v)
	case int:
		taskID = uint(v)
	default:
		return "", fmt.Errorf("invalid task_id type: %T", v)
	}

	err := tm.UpdateTaskStatus(taskID, status)
	if err != nil {
		return "", fmt.Errorf("failed to update task status: %v", err)
	}

	return fmt.Sprintf("任务状态已更新为: %s", status), nil
}

// updateTask 更新任务的多个字段（标题、内容、截止时间等）
func (e *Executor) updateTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskIDRaw := args["task_id"]
	if taskIDRaw == nil {
		return "", fmt.Errorf("task_id is required")
	}

	// 解析任务ID
	var taskID uint
	switch v := taskIDRaw.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid task_id: %s", v)
		}
		taskID = uint(id)
	case float64:
		taskID = uint(v)
	case int:
		taskID = uint(v)
	default:
		return "", fmt.Errorf("invalid task_id type: %T", v)
	}

	// 解析可选字段
	var title *string
	if titleStr, ok := args["title"].(string); ok && titleStr != "" {
		title = &titleStr
	}

	var content *string
	if contentStr, ok := args["content"].(string); ok && contentStr != "" {
		content = &contentStr
	}

	var dueTime *time.Time
	if dueTimeStr, ok := args["due_time"].(string); ok && dueTimeStr != "" {
		parsedTime, err := e.parseDueTime(dueTimeStr)
		if err != nil {
			log.Printf("WARNING: Failed to parse due_time '%s': %v\n", dueTimeStr, err)
			// 不返回错误，只是不更新截止时间
		} else {
			dueTime = &parsedTime
		}
	}

	// 更新任务
	err := tm.UpdateTask(taskID, title, content, dueTime)
	if err != nil {
		return "", fmt.Errorf("failed to update task: %v", err)
	}

	// 获取更新后的任务信息
	updatedTask, exists := tm.GetTask(taskID)
	if !exists {
		return "任务已更新", nil
	}

	return fmt.Sprintf("✅ 任务已更新！\n%s", task.FormatTaskForDisplayWithManager(updatedTask, tm)), nil
}

// getTask 获取单个任务
func (e *Executor) getTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskIDRaw := args["task_id"]
	
	// 解析任务ID
	var taskID uint
	switch v := taskIDRaw.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid task_id: %s", v)
		}
		taskID = uint(id)
	case float64:
		taskID = uint(v)
	case int:
		taskID = uint(v)
	default:
		return "", fmt.Errorf("invalid task_id type: %T", v)
	}

	t, exists := tm.GetTask(taskID)
	if !exists {
		return "", fmt.Errorf("task not found: %d", taskID)
	}

	return task.FormatTaskForDisplayWithManager(t, tm), nil
}

// deleteTask 删除任务
func (e *Executor) deleteTask(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskIDRaw := args["task_id"]
	
	// 解析任务ID
	var taskID uint
	switch v := taskIDRaw.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid task_id: %s", v)
		}
		taskID = uint(id)
	case float64:
		taskID = uint(v)
	case int:
		taskID = uint(v)
	default:
		return "", fmt.Errorf("invalid task_id type: %T", v)
	}

	err := tm.DeleteTask(taskID)
	if err != nil {
		return "", fmt.Errorf("failed to delete task: %v", err)
	}

	return fmt.Sprintf("任务 %d 已成功删除", taskID), nil
}

// updateTaskDependencies 更新任务的依赖关系
func (e *Executor) updateTaskDependencies(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	taskIDRaw := args["task_id"]
	if taskIDRaw == nil {
		return "", fmt.Errorf("task_id is required")
	}

	// 解析任务ID
	var taskID uint
	switch v := taskIDRaw.(type) {
	case string:
		id, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return "", fmt.Errorf("invalid task_id: %s", v)
		}
		taskID = uint(id)
	case float64:
		taskID = uint(v)
	case int:
		taskID = uint(v)
	default:
		return "", fmt.Errorf("invalid task_id type: %T", v)
	}

	// 解析依赖任务列表
	var dependencies []uint
	if deps, ok := args["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			var depID uint
			switch v := dep.(type) {
			case string:
				id, err := strconv.ParseUint(v, 10, 32)
				if err != nil {
					log.Printf("WARNING: Invalid dependency ID '%s': %v, skipping\n", v, err)
					continue
				}
				depID = uint(id)
			case float64:
				depID = uint(v)
			case int:
				depID = uint(v)
			default:
				log.Printf("WARNING: Invalid dependency type: %T, skipping\n", v)
				continue
			}
			dependencies = append(dependencies, depID)
		}
	}

	// 更新依赖关系
	err := tm.UpdateTaskDependencies(taskID, dependencies)
	if err != nil {
		return "", fmt.Errorf("failed to update task dependencies: %v", err)
	}

	// 获取更新后的任务信息
	updatedTask, exists := tm.GetTask(taskID)
	if !exists {
		return "✅ 任务依赖关系已更新", nil
	}

	return fmt.Sprintf("✅ 任务依赖关系已更新！\n%s", task.FormatTaskForDisplayWithManager(updatedTask, tm)), nil
}

// searchTasks 搜索任务
func (e *Executor) searchTasks(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	// 支持 keyword 和 query 两种参数名（兼容性）
	keyword, _ := args["keyword"].(string)
	if keyword == "" {
		keyword, _ = args["query"].(string)
	}
	if keyword == "" {
		// 如果都没有，列出所有任务
		return task.FormatTaskListForDisplay(tm.ListTasks("", "")), nil
	}

	// 获取所有任务并过滤
	allTasks := tm.ListTasks("", "")
	matchedTasks := make([]*task.Task, 0)

	for _, t := range allTasks {
		// 在标题和内容中搜索关键词
		if contains(t.Title, keyword) || contains(t.Content, keyword) {
			matchedTasks = append(matchedTasks, t)
		}
	}

	if len(matchedTasks) == 0 {
		return fmt.Sprintf("未找到包含 '%s' 的任务", keyword), nil
	}

	return task.FormatTaskListForDisplay(matchedTasks), nil
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// getOverdueTasks 获取过期任务
func (e *Executor) getOverdueTasks(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	overdueTasks := tm.GetOverdueTasks()

	if len(overdueTasks) == 0 {
		return "✅ 没有过期任务", nil
	}

	result := fmt.Sprintf("⚠️ 发现 %d 个过期任务：\n\n", len(overdueTasks))
	result += task.FormatTaskListForDisplay(overdueTasks)
	return result, nil
}

// getUpcomingTasks 获取即将到期的任务
func (e *Executor) getUpcomingTasks(args map[string]interface{}) (string, error) {
	tm := task.GetTaskManager()

	// 默认24小时内
	hours := 24.0
	if hoursFloat, ok := args["hours"].(float64); ok {
		hours = hoursFloat
	}

	upcomingTasks := tm.GetUpcomingTasks(time.Duration(hours) * time.Hour)

	if len(upcomingTasks) == 0 {
		return fmt.Sprintf("✅ 未来 %.0f 小时内没有即将到期的任务", hours), nil
	}

	result := fmt.Sprintf("⏰ 未来 %.0f 小时内有 %d 个即将到期的任务：\n\n", hours, len(upcomingTasks))
	result += task.FormatTaskListForDisplay(upcomingTasks)
	return result, nil
}

// GetAvailableCommands 获取可用命令列表
func (e *Executor) GetAvailableCommands() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "create_task",
			"description": "创建新任务。**重要：只有在用户明确说出'创建任务'、'记录任务'、'添加任务'等明确的任务创建指令时才使用此工具。如果用户只是分享计划、想法或讨论要做的事情（如'我要完成报告'、'明天要开会'），这是普通对话，不要使用此工具，正常回复即可。**用户明确要求创建任务时，用户说出的内容就是任务内容，从中提取标题和截止时间。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "任务具体内容（必需），这是用户说出的完整任务描述",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "任务标题（可选），从任务内容中推测一个简洁的标题，如果无法推测则留空",
					},
					"creator_id": map[string]interface{}{
						"type":        "string",
						"description": "创建任务的用户ID（必需），从上下文中的用户信息获取",
					},
					"due_time": map[string]interface{}{
						"type":        "string",
						"description": "预计结束时间（可选），从自然语言中解析，支持格式：2006-01-02 15:04:05、2006-01-02、明天、下周一等。如果用户没有提到截止时间则留空",
					},
					"dependencies": map[string]interface{}{
						"type":        "array",
						"description": "前置依赖任务ID列表（可选），任务ID是数字",
						"items": map[string]interface{}{
							"type": "number",
						},
					},
				},
				"required": []string{"content", "creator_id"},
			},
		},
		{
			"name":        "list_tasks",
			"description": "列出任务。只在用户明确询问任务列表时使用（如'我的任务'、'列出任务'、'所有任务'等）。普通聊天不使用。支持查看所有任务（团队协作）或特定用户的任务。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "任务状态筛选：pending（待处理）、in_progress（进行中）、completed（已完成）、cancelled（已取消），为空则列出所有状态的任务",
					},
					"creator_id": map[string]interface{}{
						"type":        "string",
						"description": "创建人ID筛选（可选）。如果用户说'我的任务'、'查看我的任务'，传入当前用户ID；如果用户说'所有任务'、'查看所有任务'、'团队任务'等，不传此参数或传空字符串（查看所有任务，团队协作模式）；如果用户指定查看某个人的任务，传入对应的用户ID。如果不传此参数，默认查看所有任务（团队协作模式）。",
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
			"description": "查看任务详情。只在用户明确要求查看某个任务时使用。普通聊天不使用。",
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
			"description": "更新任务状态。只在用户明确要求更新任务状态时使用（如'完成任务X'、'标记为进行中'等）。普通聊天不使用。",
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
		{
			"name":        "update_task",
			"description": "更新任务信息（标题、内容、截止时间等）。只在用户明确要求更新任务时使用。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "任务ID（必需）",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "任务标题（可选），如果要更新标题则提供此字段",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "任务内容（可选），如果要更新内容则提供此字段",
					},
					"due_time": map[string]interface{}{
						"type":        "string",
						"description": "截止时间（可选），从自然语言中解析，支持格式：2006-01-02 15:04:05、2006-01-02、明天、下周一、后天12:00等。如果要更新截止时间则提供此字段",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "delete_task",
			"description": "删除任务。只在用户明确要求删除任务时使用。注意：被依赖的任务无法删除。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "要删除的任务ID（必需）",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			"name":        "search_tasks",
			"description": "搜索任务。只在用户明确要求搜索任务时使用。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type":        "string",
						"description": "搜索关键词（必需）",
					},
				},
				"required": []string{"keyword"},
			},
		},
		{
			"name":        "get_overdue_tasks",
			"description": "获取过期任务。只在用户明确询问过期任务时使用。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "get_upcoming_tasks",
			"description": "获取即将到期的任务（默认24小时内）。只在用户明确询问即将到期的任务时使用。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"hours": map[string]interface{}{
						"type":        "number",
						"description": "时间范围（小时），默认为24",
					},
				},
			},
		},
		{
			"name":        "update_task_dependencies",
			"description": "更新任务依赖关系。只在用户明确要求更新任务依赖时使用。普通聊天不使用。",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "任务ID（必需）",
					},
					"dependencies": map[string]interface{}{
						"type":        "array",
						"description": "依赖任务ID列表（可选）",
						"items": map[string]interface{}{
							"type": "number",
						},
					},
				},
				"required": []string{"task_id"},
			},
		},
	}
}
