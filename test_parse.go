package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

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
	if !strings.Contains(content, beginMarker) {
		beginMarker = "<|tool_calls_begin|>"
		endMarker = "<|tool_calls_end|>"
		callBeginMarker = "<|tool_call_begin|>"
		callEndMarker = "<|tool_call_end|>"
		sepMarker = "<|tool_sep|>"
	}
	
	// 查找标记（精确匹配）
	beginIdx := strings.Index(content, beginMarker)
	endIdx := strings.Index(content, endMarker)
	
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		log.Printf("Could not find tool call markers: beginIdx=%d, endIdx=%d\n", beginIdx, endIdx)
		return toolCalls, cleanedContent
	}

	log.Printf("Found tool call block: beginIdx=%d, endIdx=%d\n", beginIdx, endIdx)

	// 提取工具调用部分
	toolCallsBlock := content[beginIdx+len(beginMarker) : endIdx]
	log.Printf("Tool calls block length: %d chars\n", len(toolCallsBlock))
	
	// 清理内容，移除工具调用标记
	cleanedContent = strings.TrimSpace(content[:beginIdx] + content[endIdx+len(endMarker):])

	for {
		callBeginIdx := strings.Index(toolCallsBlock, callBeginMarker)
		if callBeginIdx == -1 {
			break
		}
		
		// 找到开始标记后，查找结束标记
		remaining := toolCallsBlock[callBeginIdx:]
		callEndIdx := strings.Index(remaining, callEndMarker)
		if callEndIdx == -1 {
			break
		}
		callEndIdx += callBeginIdx

		// 提取工具调用内容
		markerEnd := callBeginIdx + len(callBeginMarker)
		markerStart := callEndIdx
		callContent := toolCallsBlock[markerEnd:markerStart]
		
		log.Printf("Extracted call content: %s\n", callContent[:min(100, len(callContent))])
		
		// 查找分隔符
		sepIdx := strings.Index(callContent, sepMarker)
		if sepIdx == -1 {
			log.Printf("Could not find separator marker in call content\n")
			toolCallsBlock = toolCallsBlock[callEndIdx+len(callEndMarker):]
			continue
		}

		// 提取工具名称和参数
		toolName := strings.TrimSpace(callContent[:sepIdx])
		argsStart := sepIdx + len(sepMarker)
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
		}

		toolCalls = append(toolCalls, map[string]interface{}{
			"name":      toolName,
			"arguments": args,
		})

		// 继续查找下一个工具调用
		toolCallsBlock = toolCallsBlock[callEndIdx+len(callEndMarker):]
	}

	return toolCalls, cleanedContent
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
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
	
	// 测试用例2: redacted格式
	test2 := `现在我为您创建依赖"写歌"任务的新任务：<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>create_task<｜tool▁sep｜>{"title": "打谱任务","content": "后天12点之前搞定打谱","due_time": "后天12:00","dependency_task_ids": [1]}<｜tool▁call▁end｜><｜tool▁calls▁end｜>`
	
	fmt.Println("=== Test 2: Redacted format ===")
	toolCalls2, cleaned2 := parseTextToolCalls(test2)
	fmt.Printf("Tool calls found: %d\n", len(toolCalls2))
	for i, tc := range toolCalls2 {
		fmt.Printf("  Tool %d: %s\n", i+1, tc["name"])
		if args, ok := tc["arguments"].(map[string]interface{}); ok {
			fmt.Printf("    Args: %v\n", args)
		}
	}
	fmt.Printf("Cleaned content: %s\n\n", cleaned2)
	
	// 测试用例3: 带空格的格式（用户实际遇到的）
	test3 := `我发现目前只有一个任务"写歌任务"，而您提到的"前两个任务"可能包括之前创建的"打谱"任务。让我先创建您要求的新任务，依赖关系将基于现有任务：<|tool_calls_begin |><|tool_call_begin | >create_task< | tool_sep | >{"creator_id":"@@73b809f17e9df57e9d2ed49289ada045245d96776c742648e7e547acb29c211c", "title": "系统测试依赖关系","content": "任务系统测试依赖关系。截止日期今天6点。依赖于前两个任务。", "due_time": "今天 18:00","dependencies": [1]]< |tool_call_end |><|tool_calls_end|>`
	
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
	
	// 测试用例4: 没有工具调用的正常文本
	test4 := `这是一个正常的回复，没有任何工具调用标记。`
	
	fmt.Println("=== Test 4: Normal text without tool calls ===")
	toolCalls4, cleaned4 := parseTextToolCalls(test4)
	fmt.Printf("Tool calls found: %d\n", len(toolCalls4))
	fmt.Printf("Cleaned content: %s\n\n", cleaned4)
}

