package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// findMarker 查找标记，支持带空格的变体（如 <|tool_calls_begin|> 或 <|tool_calls_begin |>）
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
	
	// 构建带空格的变体模式
	patterns := []string{
		"<|" + markerName + "|>",      // 标准格式
		"<|" + markerName + " |>",     // 在 | 和 > 之间有空格
		"<| " + markerName + "|>",    // 在 < 和 | 之间有空格
		"<| " + markerName + " |>",   // 两边都有空格
		"<|" + markerName + "| >",     // 在 | 和 > 之间有空格（另一种）
	}
	
	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			return idx
		}
	}
	
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 测试解析文本格式的工具调用
func parseTextToolCalls(content string) ([]map[string]interface{}, string) {
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
				toolCallsBlock = toolCallsBlock[callBeginEndIdx+callEndActualEndIdx:]
				continue
			}
		}

		toolCalls = append(toolCalls, map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		})

		// 继续查找下一个工具调用
		toolCallsBlock = toolCallsBlock[callBeginEndIdx+callEndActualEndIdx:]
	}

	return toolCalls, cleanedContent
}

func main() {
	// 测试用例3: 带空格的格式（用户实际遇到的）
	test3 := `我发现目前只有一个任务"写歌任务"，而您提到的"前两个任务"可能包括之前创建的"打谱"任务。让我先创建您要求的新任务，依赖关系将基于现有任务：<|tool_calls_begin |><|tool_call_begin | >create_task< | tool_sep | >{"creator_id":"@@73b809f17e9df57e9d2ed49289ada045245d96776c742648e7e547acb29c211c", "title": "系统测试依赖关系","content": "任务系统测试依赖关系。截止日期今天6点。依赖于前两个任务。", "due_time": "今天 18:00","dependencies": [1]}< |tool_call_end |><|tool_calls_end|>`
	
	fmt.Println("=== Test 3: Format with spaces (actual user case) ===")
	toolCalls3, cleaned3 := parseTextToolCalls(test3)
	fmt.Printf("Tool calls found: %d\n", len(toolCalls3))
	for i, tc := range toolCalls3 {
		fmt.Printf("  Tool %d: %s\n", i+1, tc["name"])
		if args, ok := tc["arguments"].(map[string]interface{}); ok {
			fmt.Printf("    Args: %v\n", args)
		}
	}
	fmt.Printf("Cleaned content: %s\n\n", cleaned3)
	
	// 测试用例1: 标准格式
	test1 := `我来为您创建一个测试依赖关系的任务。首先让我查看一下现有的任务，以便确定前两个任务的ID。<|tool_calls_begin|><|tool_call_begin|>list_tasks<|tool_sep|>{}<|tool_call_end|><|tool_calls_end|>`
	
	fmt.Println("=== Test 1: Standard format ===")
	toolCalls1, cleaned1 := parseTextToolCalls(test1)
	fmt.Printf("Tool calls found: %d\n", len(toolCalls1))
	for i, tc := range toolCalls1 {
		fmt.Printf("  Tool %d: %s\n", i+1, tc["name"])
		if args, ok := tc["arguments"].(map[string]interface{}); ok {
			fmt.Printf("    Args: %v\n", args)
		}
	}
	fmt.Printf("Cleaned content: %s\n\n", cleaned1)
}

