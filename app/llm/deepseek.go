package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/869413421/wechatbot/app/agent"
	"github.com/869413421/wechatbot/app/config"
)

// 默认 DeepSeek API 密钥（兜底）
const DefaultDeepSeekApiKey = "REPLACED_API_KEY"

// DeepSeekProvider DeepSeek 提供者实现
type DeepSeekProvider struct {
	apiKey    string
	modelName string
	baseURL   string
}

// NewDeepSeekProvider 创建 DeepSeek 提供者
func NewDeepSeekProvider() *DeepSeekProvider {
	cfg := config.LoadConfig()
	modelName := cfg.ModelName
	if modelName == "" {
		modelName = "deepseek-chat"
	}

	// 使用配置的 API 密钥，如果为空则使用默认密钥作为兜底
	apiKey := cfg.ApiKey
	if apiKey == "" {
		apiKey = DefaultDeepSeekApiKey
		log.Printf("Using default DeepSeek API key as fallback\n")
	}

	return &DeepSeekProvider{
		apiKey:    apiKey,
		modelName: modelName,
		baseURL:   "https://api.deepseek.com/v1/chat/completions",
	}
}

// Chat 发送聊天请求（支持 Agent 功能）
func (p *DeepSeekProvider) Chat(messages []Message) (string, error) {
	// 获取可用工具
	executor := agent.NewExecutor()
	tools := executor.GetAvailableCommands()

	requestBody := map[string]interface{}{
		"model":    p.modelName,
		"messages": messages,
		"tools":    p.formatTools(tools),
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	log.Printf("request DeepSeek %s json string : %v", p.modelName, string(requestData))
	req, err := http.NewRequest("POST", p.baseURL, bytes.NewBuffer(requestData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	// 检查 HTTP 状态码
	if response.StatusCode != http.StatusOK {
		log.Printf("DeepSeek API error: status %d, body: %s\n", response.StatusCode, string(body))
		return "", fmt.Errorf("DeepSeek API error: status %d", response.StatusCode)
	}

	var responseBody struct {
		Choices []struct {
			Message struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				ToolCalls []struct {
					Id       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	log.Println(string(body))
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	// 检查 API 错误
	if responseBody.Error.Message != "" {
		log.Printf("DeepSeek API error: %s (type: %s)\n", responseBody.Error.Message, responseBody.Error.Type)
		return "", fmt.Errorf("DeepSeek API error: %s", responseBody.Error.Message)
	}

	if len(responseBody.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	message := responseBody.Choices[0].Message

	// 检查是否有工具调用
	if len(message.ToolCalls) > 0 {
		log.Printf("DeepSeek requested tool calls: %d\n", len(message.ToolCalls))

		// 执行工具调用
		var toolResults []Message
		for i, toolCall := range message.ToolCalls {
			log.Printf("Processing tool call %d/%d: %s\n", i+1, len(message.ToolCalls), toolCall.Function.Name)
			
			// 使用 defer recover 捕获 panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("PANIC in tool execution: %v\n", r)
						toolResults = append(toolResults, Message{
							Role:    "tool",
							Content: fmt.Sprintf("Error: panic occurred: %v", r),
						})
					}
				}()
				
				// 解析参数
				var args map[string]interface{}
				if toolCall.Function.Arguments != "" {
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
						log.Printf("Failed to parse tool arguments: %v, raw: %s\n", err, toolCall.Function.Arguments)
						// 添加错误结果
						toolResults = append(toolResults, Message{
							Role:    "tool",
							Content: fmt.Sprintf("Error: failed to parse arguments: %v", err),
						})
						return
					}
				} else {
					args = make(map[string]interface{})
				}
				
				log.Printf("Tool arguments parsed: %v\n", args)

				// 执行命令
				log.Printf("Executing tool: %s\n", toolCall.Function.Name)
				result, err := executor.ExecuteCommand(toolCall.Function.Name, args)
				if err != nil {
					log.Printf("Tool execution failed: %v\n", err)
					toolResults = append(toolResults, Message{
						Role:    "tool",
						Content: fmt.Sprintf("Error: %v", err),
					})
				} else {
					log.Printf("Tool execution success, result length: %d\n", len(result))
					if len(result) > 200 {
						log.Printf("Tool result (first 200 chars): %s...\n", result[:200])
					} else {
						log.Printf("Tool result: %s\n", result)
					}
					toolResults = append(toolResults, Message{
						Role:    "tool",
						Content: result,
					})
				}
			}()
		}
		
		log.Printf("All tool calls processed, results count: %d\n", len(toolResults))

		// 如果有工具调用结果，需要再次调用 API 获取最终回复
		if len(toolResults) > 0 {
			// 构建新的消息列表（使用原始格式，包含 tool_calls）
			newMessagesRaw := make([]map[string]interface{}, 0, len(messages)+1+len(toolResults))
			
			// 添加历史消息
			for _, msg := range messages {
				newMessagesRaw = append(newMessagesRaw, map[string]interface{}{
					"role":    msg.Role,
					"content": msg.Content,
				})
			}

			// 添加 assistant 的工具调用消息（包含 tool_calls 字段）
			assistantMsgRaw := map[string]interface{}{
				"role":    "assistant",
				"content": message.Content,
			}
			if len(message.ToolCalls) > 0 {
				assistantMsgRaw["tool_calls"] = message.ToolCalls
			}
			newMessagesRaw = append(newMessagesRaw, assistantMsgRaw)

			// 添加工具结果消息（需要包含 tool_call_id）
			for i, toolCall := range message.ToolCalls {
				if i < len(toolResults) {
					toolMsg := map[string]interface{}{
						"role":         "tool",
						"content":      toolResults[i].Content,
						"tool_call_id": toolCall.Id,
					}
					newMessagesRaw = append(newMessagesRaw, toolMsg)
				}
			}

			log.Printf("Calling DeepSeek again with tool results, messages count: %d\n", len(newMessagesRaw))
			log.Printf("Tool results count: %d\n", len(toolResults))
			for i, tr := range toolResults {
				log.Printf("Tool result %d: %s\n", i, tr.Content)
			}

			// 再次调用 API 获取最终回复（不包含 tools）
			finalReply, err := p.chatWithRawMessages(newMessagesRaw)
			if err != nil {
				log.Printf("ERROR: Failed to get final reply after tool execution: %v\n", err)
				// 如果获取最终回复失败，直接返回工具执行结果
				if len(toolResults) > 0 {
					log.Printf("Returning tool execution result as fallback: %s\n", toolResults[0].Content)
					return toolResults[0].Content, nil
				}
				log.Printf("ERROR: No tool results to return\n")
				return "", err
			}
			
			if finalReply == "" {
				log.Printf("WARNING: Final reply is empty after tool execution\n")
				// 如果最终回复为空，返回工具执行结果
				if len(toolResults) > 0 {
					log.Printf("Returning tool execution result as fallback: %s\n", toolResults[0].Content)
					return toolResults[0].Content, nil
				}
				log.Printf("Returning default message\n")
				return "任务已处理完成", nil
			}
			
			log.Printf("SUCCESS: Got final reply from DeepSeek: %s\n", finalReply)
			return finalReply, nil
		}
	}

	reply := message.Content
	if reply == "" && len(message.ToolCalls) == 0 {
		return "", fmt.Errorf("empty response from DeepSeek")
	}
	
	log.Printf("DeepSeek response text: %s \n", reply)
	return reply, nil
}

// chatWithRawMessages 使用原始消息格式发送聊天请求（不包含工具，用于获取最终回复）
func (p *DeepSeekProvider) chatWithRawMessages(messages []map[string]interface{}) (string, error) {
	requestBody := map[string]interface{}{
		"model":    p.modelName,
		"messages": messages,
	}

	requestData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	log.Printf("request DeepSeek %s (final response) json string : %v", p.modelName, string(requestData))
	req, err := http.NewRequest("POST", p.baseURL, bytes.NewBuffer(requestData))
	if err != nil {
		log.Printf("ERROR: Failed to create request: %v\n", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to send request: %v\n", err)
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("ERROR: Failed to read response body: %v\n", err)
		return "", err
	}

	log.Printf("DeepSeek final response status: %d, body: %s\n", response.StatusCode, string(body))

	if response.StatusCode != http.StatusOK {
		log.Printf("ERROR: DeepSeek API error: status %d, body: %s\n", response.StatusCode, string(body))
		return "", fmt.Errorf("DeepSeek API error: status %d", response.StatusCode)
	}

	var responseBody struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	log.Println(string(body))
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if responseBody.Error.Message != "" {
		return "", fmt.Errorf("DeepSeek API error: %s", responseBody.Error.Message)
	}

	if len(responseBody.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	reply := responseBody.Choices[0].Message.Content
	if reply == "" {
		return "", fmt.Errorf("empty response content")
	}
	
	log.Printf("DeepSeek final response text: %s \n", reply)
	return reply, nil
}

// formatTools 格式化工具定义
func (p *DeepSeekProvider) formatTools(tools []map[string]interface{}) []map[string]interface{} {
	formatted := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		formatted = append(formatted, map[string]interface{}{
			"type":     "function",
			"function": tool,
		})
	}
	return formatted
}

// GetModelName 获取模型名称
func (p *DeepSeekProvider) GetModelName() string {
	return p.modelName
}

// GetBaseURL 获取 API 端点
func (p *DeepSeekProvider) GetBaseURL() string {
	return p.baseURL
}
