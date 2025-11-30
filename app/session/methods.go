package session

import (
	"log"

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
		systemMsg := `你是一个智能助手，具备 Agent 能力，可以执行任务管理功能。

你的主要能力包括：
1. **任务管理**：
   - 创建任务：当用户说"记录任务"时，帮助用户创建任务，记录任务标题、内容、截止时间、依赖关系等
   - 查询任务：可以列出所有任务、按状态筛选、查看任务详情、统计任务数量
   - 更新任务：可以更新任务状态（待处理、进行中、已完成、已取消）
   - 任务会自动保存到本地文件，支持依赖管理，防止循环依赖

2. **对话能力**：
   - 回答用户问题
   - 提供帮助和建议
   - 支持上下文记忆

当用户询问你的能力时，请主动介绍你的任务管理功能，并说明如何使用。`
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


