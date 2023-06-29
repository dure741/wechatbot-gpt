package gtp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/869413421/wechatbot/config"
)

//chat/completions
const BASEURL = "https://api.openai.com/v1/chat/completions"

// ChatGPTResponseBody 请求体
// {
// 	"id": "chatcmpl-7WEbrkOefOK2dH4hNgW5l3YCuxKpB",
// 	"object": "chat.completion",
// 	"created": 1687917011,
// 	"model": "gpt-3.5-turbo-0613",
// 	"choices": [
// 	  {
// 		"index": 0,
// 		"message": {
// 		  "role": "assistant",
// 		  "content": "你好！有什么我可以帮助你的吗？"
// 		},
// 		"finish_reason": "stop"
// 	  }
// 	],
// 	"usage": {
// 	  "prompt_tokens": 30,
// 	  "completion_tokens": 18,
// 	  "total_tokens": 48
// 	}
// }
type ChatGPTResponseBody struct {
	Id      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []ChoiceItem `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ChoiceItem struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var sessionMap = make(map[string][]Message)

// Completions gtp文本模型回复
// curl --http1.1 https://api.openai.com/v1/chat/completions \
//   -H "Content-Type: application/json" \
//   -H "Authorization: Bearer $OPENAI_API_KEY" \
//   -d '{
//      "model": "gpt-3.5-turbo",
//      "messages": [{"role": "user", "content": "你好"}]
//    }'
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

	requestBody := ChatGPTRequestBody{
		Model:    "gpt-3.5-turbo",
		Messages: getSession(sessionId),
	}
	requestData, err := json.Marshal(requestBody)

	if err != nil {
		return "", err
	}
	log.Printf("request gtp json string : %v", string(requestData))
	req, err := http.NewRequest("POST", BASEURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", err
	}

	apiKey := config.LoadConfig().ApiKey
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	gptResponseBody := &ChatGPTResponseBody{}
	log.Println(string(body))
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		return "", err
	}
	var reply string
	if len(gptResponseBody.Choices) > 0 {
		for _, v := range gptResponseBody.Choices {
			reply = v.Message.Content
			addSession(sessionId, v.Message)
			break
		}
	}
	log.Printf("gpt response text: %s \n", reply)
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
		sessionMap[sessionId] = append(sessionMap[sessionId], Message{Role: "system", Content: "你是一个非常有帮助的聊天机器人"})
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
