package session

import (
	"fmt"
	"log"
	"strings"

	"github.com/869413421/wechatbot/app/config"
	"github.com/869413421/wechatbot/app/llm"
)

// Message 消息结构（使用 llm 包的 Message）
type Message = llm.Message

var sessionMap = make(map[string][]Message)

// Completions 会话完成处理（支持多种 AI 模型）
func Completions(sessionId, msg string, change_str string) (string, error) {
	if change_str != "" {
		changeRoleAction(sessionId, change_str)
	}
	if msg == "换个话题" || msg == "换个话题吧" || "清空" == msg || "清空对话" == msg {
		clearSession(sessionId)
	}
	if msg == "get:role" {
		return "role: " + getSystemMsg(sessionId), nil
	}
	if msg == "get:session" {
		return getSessionMsg(sessionId), nil
	}

	addSession(sessionId, Message{Role: "user", Content: msg})

	// 根据配置创建 AI 提供者
	provider := llm.NewProvider()

	// 获取会话历史
	messages := getSession(sessionId)

	// 调用 AI 提供者
	reply, err := provider.Chat(messages)
	if err != nil {
		log.Printf("AI request error: %v \n", err)
		// 即使出错，也返回友好的错误提示
		errorMsg := "抱歉，处理您的请求时出现了问题，请稍后再试。"
		addSession(sessionId, Message{Role: "assistant", Content: errorMsg})
		return errorMsg, nil
	}

	// 检查回复是否为空
	if reply == "" {
		log.Printf("AI returned empty reply\n")
		reply = "抱歉，我暂时无法处理这个请求，请稍后再试。"
	}

	// 将 AI 回复添加到会话
	if reply != "" {
		addSession(sessionId, Message{Role: "assistant", Content: reply})
	}

	log.Printf("AI response text: %s \n", reply)
	return reply, nil
}

func addSession(sessionId string, msg Message) {
	session := getSession(sessionId)
	session = append(session, msg)
	if len(session) > config.LoadConfig().MaxMsg*2+1 {
		// 删除除了"system"的最早的一条对话消息
		session = append(session[:1], session[3:]...)
	}
	sessionMap[sessionId] = session
}

func getSession(sessionId string) []Message {
	if _, ok := sessionMap[sessionId]; !ok {
		sessionMap[sessionId] = make([]Message, 0)
		// 从sessionId中提取用户ID（格式：NickName-UserName，使用UserName部分）
		userID := extractUserIDFromSessionId(sessionId)
		systemMsg := fmt.Sprintf(`你是一个友好的AI助手，可以自然对话、回答问题、提供建议。

当前用户ID: %s

如果你需要管理任务，我也可以帮忙。当用户明确提到任务相关需求时（比如"创建任务"、"查看任务"、"我的任务"等），可以使用任务管理工具。

任务管理使用说明：
- 创建任务时，使用 create_task 工具，creator_id 使用: %s
- 如果用户提到时间（如"今天13点"、"明天12点"、"后天下午4点"），需要将自然语言转换为标准格式 "YYYY-MM-DD HH:MM:SS" 再传递给 due_time 参数
- 时间转换示例："今天13点" → 当前日期 + " 13:00:00"，"明天12点" → 明天日期 + " 12:00:00"
- 其他工具按需使用：list_tasks（列出任务）、get_task（查看任务详情）、update_task（更新任务）、update_task_dependencies（更新依赖）等

普通聊天时就像普通AI助手一样自然回复，不需要调用任何工具。`, userID, userID)
		sessionMap[sessionId] = append(sessionMap[sessionId], Message{Role: "system", Content: systemMsg})
	}
	return sessionMap[sessionId]
}

func changeRoleAction(sessionId string, msg string) {
	session := getSession(sessionId)
	if len(session) > 0 {
		sessionMap[sessionId][0] = Message{Role: "system", Content: msg}
	} else {
		sessionMap[sessionId] = append(session, Message{Role: "system", Content: msg})
	}
}

// 清空除了"system"的所有对话消息
func clearSession(sessionId string) {
	sessionMap[sessionId] = sessionMap[sessionId][:1]
}

// 获得role为"system"的消息
func getSystemMsg(sessionId string) string {
	session := getSession(sessionId)
	if len(session) > 0 {
		return session[0].Content
	}
	return ""
}

// 获取session的所有消息
func getSessionMsg(sessionId string) string {
	session := getSession(sessionId)
	var msg string
	for _, v := range session {
		if v.Role == "system" {
			continue
		}
		if v.Role == "user" {
			msg += "你: " + v.Content + "\n"
		} else {
			msg += "机器人: " + v.Content + "\n"
		}
	}
	return msg
}

// extractUserIDFromSessionId 从sessionId中提取用户ID
// sessionId格式：NickName-UserName，返回UserName部分作为用户ID
func extractUserIDFromSessionId(sessionId string) string {
	parts := strings.Split(sessionId, "-")
	if len(parts) >= 2 {
		// 返回UserName部分（去掉NickName）
		return strings.Join(parts[1:], "-")
	}
	// 如果格式不对，返回整个sessionId
	return sessionId
}


