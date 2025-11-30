package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

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
	// 从消息中提取用户ID
	userID := p.extractUserIDFromMessages(messages)
	return p.ChatWithUserID(messages, userID)
}

// ChatWithUserID 发送聊天请求（支持 Agent 功能，带用户ID）
func (p *DeepSeekProvider) ChatWithUserID(messages []Message, userID string) (string, error) {
	// 从sessionId中提取用户ID（格式：NickName-UserName，使用UserName部分）
	if userID == "" {
		userID = p.extractUserIDFromMessages(messages)
	}
	
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

	// 检查是否有文本格式的工具调用（DeepSeek 可能返回多种格式）
	if strings.Contains(message.Content, "redacted_tool_calls") || strings.Contains(message.Content, "tool_calls_begin") {
		log.Printf("Detected tool calls in content, parsing...\n")
		toolCalls, cleanedContent := p.parseTextToolCalls(message.Content)
		if len(toolCalls) > 0 {
			log.Printf("Parsed %d tool calls from text format\n", len(toolCalls))
			// 将解析出的工具调用转换为标准格式并执行
			return p.executeRedactedToolCalls(messages, toolCalls, cleanedContent, userID)
		}
		// 如果没有解析出工具调用，清理内容中的标记
		message.Content = cleanedContent
	}

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

				// 如果是create_task命令，自动注入creator_id
				if toolCall.Function.Name == "create_task" {
					if _, exists := args["creator_id"]; !exists || args["creator_id"] == "" {
						args["creator_id"] = userID
						log.Printf("Auto-injected creator_id: %s\n", userID)
					}
				}

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
			finalReply, err := p.chatWithRawMessagesAndUserID(newMessagesRaw, userID)
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
			
			// 检查最终回复中是否还包含工具调用标记（说明之前的解析可能失败了）
			if strings.Contains(finalReply, "redacted_tool_calls") || strings.Contains(finalReply, "tool_calls_begin") {
				log.Printf("WARNING: Final reply still contains tool call markers, attempting to parse again...\n")
				toolCalls, cleanedReply := p.parseTextToolCalls(finalReply)
				if len(toolCalls) > 0 {
					log.Printf("Found %d tool calls in final reply, executing...\n", len(toolCalls))
					// 执行工具调用并获取新的回复
					return p.executeRedactedToolCalls(messages, toolCalls, cleanedReply, userID)
				}
				// 如果解析失败，至少清理标记
				finalReply = cleanedReply
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
	return p.chatWithRawMessagesAndUserID(messages, "")
}

// chatWithRawMessagesAndUserID 使用原始消息格式发送聊天请求（带用户ID）
func (p *DeepSeekProvider) chatWithRawMessagesAndUserID(messages []map[string]interface{}, userID string) (string, error) {
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
	
	// 检查回复中是否包含工具调用标记（可能在最终回复中出现）
	if strings.Contains(reply, "redacted_tool_calls") || strings.Contains(reply, "tool_calls_begin") {
		log.Printf("WARNING: Final reply contains tool call markers, attempting to parse and execute...\n")
		toolCalls, cleanedReply := p.parseTextToolCalls(reply)
		if len(toolCalls) > 0 {
			log.Printf("Found %d tool calls in final reply, executing...\n", len(toolCalls))
			// 如果userID为空，尝试从消息中提取
			if userID == "" {
				// 从消息中提取userID（需要将map转换为Message格式）
				msgList := make([]Message, 0)
				for _, msg := range messages {
					if role, ok := msg["role"].(string); ok {
						if content, ok2 := msg["content"].(string); ok2 {
							msgList = append(msgList, Message{Role: role, Content: content})
						}
					}
				}
				userID = p.extractUserIDFromMessages(msgList)
			}
			// 执行工具调用
			msgList := make([]Message, 0)
			for _, msg := range messages {
				if role, ok := msg["role"].(string); ok {
					if content, ok2 := msg["content"].(string); ok2 {
						msgList = append(msgList, Message{Role: role, Content: content})
					}
				}
			}
			return p.executeRedactedToolCalls(msgList, toolCalls, cleanedReply, userID)
		}
		// 如果解析失败，至少清理标记
		reply = cleanedReply
		log.Printf("Cleaned reply: %s\n", reply[:min(200, len(reply))])
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

// extractUserIDFromMessages 从消息中提取用户ID
func (p *DeepSeekProvider) extractUserIDFromMessages(messages []Message) string {
	// 从系统消息中提取用户信息
	// 系统消息可能包含"当前用户ID: xxx"格式的信息
	for _, msg := range messages {
		if msg.Role == "system" {
			// 尝试从系统消息中提取用户ID
			// 格式：当前用户ID: xxx 或 UserID: xxx
			if idx := strings.Index(msg.Content, "当前用户ID:"); idx != -1 {
				userID := strings.TrimSpace(msg.Content[idx+len("当前用户ID:"):])
				if idx2 := strings.Index(userID, "\n"); idx2 != -1 {
					userID = userID[:idx2]
				}
				return strings.TrimSpace(userID)
			}
		}
	}
	return ""
}

// findMarker 查找标记，支持带空格的变体（如 <|tool_calls_begin|> 或 <|tool_calls_begin | >）
func findMarker(content, baseMarker string) int {
	// 先尝试精确匹配
	if idx := strings.Index(content, baseMarker); idx != -1 {
		return idx
	}
	
	// 提取标记名称（去掉 <| 和 |>）
	// baseMarker 格式: <|tool_calls_begin|>
	if !strings.HasPrefix(baseMarker, "<|") || !strings.HasSuffix(baseMarker, ">") {
		return -1
	}
	
	// 提取中间部分: tool_calls_begin| 或 tool_calls_begin
	inner := baseMarker[2 : len(baseMarker)-1] // 去掉 <| 和 >
	// 去掉最后的 |（如果有）
	markerName := strings.TrimSuffix(inner, "|")
	markerName = strings.TrimSpace(markerName)
	
	// 构建带空格的变体模式（支持各种空格组合）
	// DeepSeek 可能返回各种格式，如：
	// - <|tool_calls_begin|> (标准)
	// - <|tool_calls_begin |> (| 和 > 之间有空格)
	// - < |tool_calls_begin|> (< 和 | 之间有空格)
	// - < | tool_calls_begin | > (所有位置都有空格)
	patterns := []string{
		"<|" + markerName + "|>",       // 标准格式: <|tool_calls_begin|>
		"<|" + markerName + " |>",       // 在 | 和 > 之间有空格: <|tool_calls_begin |>
		"<| " + markerName + "|>",        // 在 < 和 | 之间有空格: <| tool_calls_begin|>
		"<| " + markerName + " |>",       // <| 和 |> 之间都有空格: <| tool_calls_begin |>
		"<|" + markerName + "| >",       // 在 | 和 > 之间有空格: <|tool_calls_begin| >
		"<|" + markerName + " | >",       // 在 | 和 > 之间都有空格: <|tool_calls_begin | >
		"<| " + markerName + "| >",       // 另一种组合: <| tool_calls_begin| >
		"<| " + markerName + " | >",      // 所有位置都有空格: <| tool_calls_begin | >
		"< |" + markerName + "|>",        // < 和 | 之间有空格: < |tool_calls_begin|>
		"< |" + markerName + " |>",       // < 和 | 之间有空格，| 和 > 之间也有: < |tool_calls_begin |>
		"< |" + markerName + "| >",       // < 和 | 之间有空格，| 和 > 之间也有: < |tool_calls_begin| >
		"< |" + markerName + " | >",      // 所有位置都有空格: < |tool_calls_begin | >
		"< | " + markerName + "|>",        // < 和 | 之间，| 和 markerName 之间都有空格: < | tool_calls_begin|>
		"< | " + markerName + " |>",      // 更多空格组合: < | tool_calls_begin |>
		"< | " + markerName + "| >",      // 更多空格组合: < | tool_calls_begin| >
		"< | " + markerName + " | >",     // 所有位置都有空格: < | tool_calls_begin | >
	}
	
	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			return idx
		}
	}
	
	return -1
}

// parseTextToolCalls 解析文本格式的工具调用（支持多种格式，包括带空格的变体）
func (p *DeepSeekProvider) parseTextToolCalls(content string) ([]map[string]interface{}, string) {
	toolCalls := make([]map[string]interface{}, 0)
	cleanedContent := content

	// 尝试解析 redacted_tool_calls 格式
	beginMarker := "<｜tool▁calls▁begin｜>"
	endMarker := "<｜tool▁calls▁end｜>"
	callBeginMarker := "<｜tool▁call▁begin｜>"
	callEndMarker := "<｜tool▁call▁end｜>"
	sepMarker := "<｜tool▁sep｜>"
	
	// 如果没找到 redacted 格式，尝试 tool_calls 格式
	beginIdx := findMarker(content, beginMarker)
	if beginIdx == -1 {
		beginMarker = "<|tool_calls_begin|>"
		endMarker = "<|tool_calls_end|>"
		callBeginMarker = "<|tool_call_begin|>"
		callEndMarker = "<|tool_call_end|>"
		sepMarker = "<|tool_sep|>"
		beginIdx = findMarker(content, beginMarker)
	}
	
	// 查找标记（支持带空格的变体）
	endIdx := findMarker(content, endMarker)
	
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		log.Printf("Could not find tool call markers: beginIdx=%d, endIdx=%d\n", beginIdx, endIdx)
		return toolCalls, cleanedContent
	}

	log.Printf("Found tool call block: beginIdx=%d, endIdx=%d\n", beginIdx, endIdx)

	// 需要找到实际的结束位置（可能包含空格）
	// 从 beginIdx 开始查找，找到第一个匹配的结束标记
	actualEndIdx := endIdx
	// 尝试找到结束标记的实际结束位置（考虑可能的空格）
	for actualEndIdx < len(content) {
		if content[actualEndIdx] == '>' {
			// 找到结束标记的结束位置
			actualEndIdx++
			break
		}
		actualEndIdx++
		if actualEndIdx >= len(content) {
			break
		}
	}
	
	// 提取工具调用部分（从开始标记的结束到结束标记的开始）
	// 需要找到开始标记的实际结束位置
	actualBeginEndIdx := beginIdx
	for actualBeginEndIdx < len(content) && actualBeginEndIdx < endIdx {
		if content[actualBeginEndIdx] == '>' {
			actualBeginEndIdx++
			break
		}
		actualBeginEndIdx++
	}
	
	toolCallsBlock := content[actualBeginEndIdx:endIdx]
	log.Printf("Tool calls block length: %d chars\n", len(toolCallsBlock))
	
	// 清理内容，移除工具调用标记（从开始标记的开始到结束标记的结束）
	cleanedContent = strings.TrimSpace(content[:beginIdx] + content[actualEndIdx:])

	for {
		callBeginIdx := findMarker(toolCallsBlock, callBeginMarker)
		if callBeginIdx == -1 {
			break
		}
		
		// 找到开始标记的实际结束位置
		callBeginEndIdx := callBeginIdx
		for callBeginEndIdx < len(toolCallsBlock) {
			if toolCallsBlock[callBeginEndIdx] == '>' {
				callBeginEndIdx++
				break
			}
			callBeginEndIdx++
		}
		
		// 找到开始标记后，查找结束标记
		remaining := toolCallsBlock[callBeginEndIdx:]
		callEndIdx := findMarker(remaining, callEndMarker)
		if callEndIdx == -1 {
			break
		}
		
		// 找到结束标记的实际结束位置
		callEndActualEndIdx := callEndIdx
		for callEndActualEndIdx < len(remaining) {
			if remaining[callEndActualEndIdx] == '>' {
				callEndActualEndIdx++
				break
			}
			callEndActualEndIdx++
		}
		
		// 提取工具调用内容（从开始标记结束到结束标记开始）
		callContent := remaining[:callEndIdx]
		
		log.Printf("Extracted call content: %s\n", callContent[:min(100, len(callContent))])
		
		// 查找分隔符（支持带空格的变体）
		sepIdx := findMarker(callContent, sepMarker)
		if sepIdx == -1 {
			log.Printf("Could not find separator marker in call content\n")
			toolCallsBlock = toolCallsBlock[callBeginEndIdx+callEndActualEndIdx:]
			continue
		}
		
		// 找到分隔符的实际结束位置
		sepEndIdx := sepIdx
		for sepEndIdx < len(callContent) {
			if callContent[sepEndIdx] == '>' {
				sepEndIdx++
				break
			}
			sepEndIdx++
		}

		// 提取工具名称和参数
		toolName := strings.TrimSpace(callContent[:sepIdx])
		argsStart := sepEndIdx
		argsStr := strings.TrimSpace(callContent[argsStart:])
		
		log.Printf("Parsing tool: %s, args: %s\n", toolName, argsStr[:min(200, len(argsStr))])

		// 解析参数JSON
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			log.Printf("Failed to parse tool arguments: %v, raw: %s\n", err, argsStr)
			// 尝试修复常见的JSON问题
			argsStrFixed := strings.ReplaceAll(argsStr, "\n", " ")
			argsStrFixed = strings.ReplaceAll(argsStrFixed, "\t", " ")
			// 移除多余的空格
			for strings.Contains(argsStrFixed, "  ") {
				argsStrFixed = strings.ReplaceAll(argsStrFixed, "  ", " ")
			}
			if err2 := json.Unmarshal([]byte(argsStrFixed), &args); err2 != nil {
				log.Printf("Failed to parse tool arguments after fix: %v\n", err2)
				toolCallsBlock = toolCallsBlock[callEndIdx+len(callEndMarker):]
				continue
			}
			log.Printf("Successfully parsed after fix\n")
		}

		// 处理参数名称转换（dependency_task_ids -> dependencies）
		if deps, ok := args["dependency_task_ids"].([]interface{}); ok {
			args["dependencies"] = deps
			delete(args, "dependency_task_ids")
		}

		toolCall := map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		}
		toolCalls = append(toolCalls, toolCall)
		log.Printf("Parsed tool call: %s with args: %v\n", toolName, args)

		// 继续查找下一个工具调用
		toolCallsBlock = toolCallsBlock[callEndIdx+len(callEndMarker):]
	}

	log.Printf("Parsed %d tool calls, cleaned content: %s\n", len(toolCalls), cleanedContent)
	return toolCalls, cleanedContent
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// executeRedactedToolCalls 执行解析出的工具调用
func (p *DeepSeekProvider) executeRedactedToolCalls(messages []Message, toolCalls []map[string]interface{}, cleanedContent string, userID string) (string, error) {
	executor := agent.NewExecutor()
	var toolResults []Message

	for i, toolCall := range toolCalls {
		toolName, _ := toolCall["name"].(string)
		args, _ := toolCall["arguments"].(map[string]interface{})

		if toolName == "" {
			continue
		}

		log.Printf("Executing redacted tool call %d/%d: %s\n", i+1, len(toolCalls), toolName)

		// 如果是create_task命令，自动注入creator_id
		if toolName == "create_task" {
			if _, exists := args["creator_id"]; !exists || args["creator_id"] == "" {
				args["creator_id"] = userID
				log.Printf("Auto-injected creator_id: %s\n", userID)
			}
		}

		// 执行命令
		result, err := executor.ExecuteCommand(toolName, args)
		if err != nil {
			log.Printf("Tool execution failed: %v\n", err)
			toolResults = append(toolResults, Message{
				Role:    "tool",
				Content: fmt.Sprintf("Error: %v", err),
			})
		} else {
			log.Printf("Tool execution success: %s\n", result)
			toolResults = append(toolResults, Message{
				Role:    "tool",
				Content: result,
			})
		}
	}

	// 如果有工具执行结果，再次调用API获取最终回复
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

		// 添加清理后的assistant消息（不包含工具调用）
		if cleanedContent != "" {
			newMessagesRaw = append(newMessagesRaw, map[string]interface{}{
				"role":    "assistant",
				"content": cleanedContent,
			})
		}

		// 添加工具结果（需要包含 tool_call_id，但redacted格式没有，使用序号）
		for i, tr := range toolResults {
			toolMsg := map[string]interface{}{
				"role":    "tool",
				"content": tr.Content,
			}
			// 尝试从工具调用中获取ID（如果有）
			if i < len(toolCalls) {
				if toolID, ok := toolCalls[i]["id"].(string); ok && toolID != "" {
					toolMsg["tool_call_id"] = toolID
				}
			}
			newMessagesRaw = append(newMessagesRaw, toolMsg)
		}

		// 再次调用API获取最终回复（不包含 tools）
		finalReply, err := p.chatWithRawMessagesAndUserID(newMessagesRaw, userID)
		if err != nil {
			log.Printf("ERROR: Failed to get final reply after tool execution: %v\n", err)
			// 如果获取最终回复失败，返回工具执行结果
			if len(toolResults) > 0 {
				return toolResults[0].Content, nil
			}
			return "", err
		}

		if finalReply == "" {
			if len(toolResults) > 0 {
				return toolResults[0].Content, nil
			}
			return "任务已处理完成", nil
		}

		// 再次检查最终回复中是否还包含工具调用标记
		if strings.Contains(finalReply, "redacted_tool_calls") || strings.Contains(finalReply, "tool_calls_begin") {
			log.Printf("WARNING: Final reply from executeRedactedToolCalls still contains tool call markers, parsing again...\n")
			toolCalls2, cleanedReply2 := p.parseTextToolCalls(finalReply)
			if len(toolCalls2) > 0 {
				log.Printf("Found %d tool calls in final reply, executing recursively...\n", len(toolCalls2))
				// 递归执行（但限制递归深度，避免无限循环）
				// 构建新的消息列表用于递归（从原始messages开始）
				recursiveMessages := make([]Message, 0, len(messages)+1+len(toolResults))
				recursiveMessages = append(recursiveMessages, messages...)
				if cleanedContent != "" {
					recursiveMessages = append(recursiveMessages, Message{
						Role:    "assistant",
						Content: cleanedContent,
					})
				}
				recursiveMessages = append(recursiveMessages, toolResults...)
				return p.executeRedactedToolCalls(recursiveMessages, toolCalls2, cleanedReply2, userID)
			}
			// 如果解析失败，至少清理标记
			finalReply = cleanedReply2
		}

		// 再次检查最终回复中是否还包含工具调用标记
		if strings.Contains(finalReply, "redacted_tool_calls") || strings.Contains(finalReply, "tool_calls_begin") {
			log.Printf("WARNING: Final reply from executeRedactedToolCalls still contains tool call markers, parsing again...\n")
			toolCalls2, cleanedReply2 := p.parseTextToolCalls(finalReply)
			if len(toolCalls2) > 0 {
				log.Printf("Found %d tool calls in final reply, executing recursively...\n", len(toolCalls2))
				// 递归执行（但限制递归深度，避免无限循环）
				// 构建新的消息列表用于递归（从原始messages开始）
				recursiveMessages := make([]Message, 0, len(messages)+1+len(toolResults))
				recursiveMessages = append(recursiveMessages, messages...)
				if cleanedContent != "" {
					recursiveMessages = append(recursiveMessages, Message{
						Role:    "assistant",
						Content: cleanedContent,
					})
				}
				recursiveMessages = append(recursiveMessages, toolResults...)
				return p.executeRedactedToolCalls(recursiveMessages, toolCalls2, cleanedReply2, userID)
			}
			// 如果解析失败，至少清理标记
			finalReply = cleanedReply2
			log.Printf("Cleaned final reply: %s\n", finalReply[:min(200, len(finalReply))])
		}

		return finalReply, nil
	}

	// 如果没有工具执行结果，返回清理后的内容
	return cleanedContent, nil
}
