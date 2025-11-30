package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// findMarker 查找标记，支持带空格的变体
func findMarker(content, baseMarker string) int {
	if idx := strings.Index(content, baseMarker); idx != -1 {
		return idx
	}
	
	if !strings.HasPrefix(baseMarker, "<|") || !strings.HasSuffix(baseMarker, ">") {
		return -1
	}
	
	inner := baseMarker[2 : len(baseMarker)-1]
	markerName := strings.TrimSuffix(inner, "|")
	markerName = strings.TrimSpace(markerName)
	
	patterns := []string{
		"<|" + markerName + "|>",
		"<|" + markerName + " |>",
		"<| " + markerName + "|>",
		"<| " + markerName + " |>",
		"<|" + markerName + "| >",
		"<|" + markerName + " | >",
		"<| " + markerName + "| >",
		"<| " + markerName + " | >",
		"< |" + markerName + "|>",
		"< |" + markerName + " |>",
		"< |" + markerName + "| >",
		"< |" + markerName + " | >",
	}
	
	fmt.Printf("DEBUG findMarker: baseMarker=%s, markerName=%s\n", baseMarker, markerName)
	fmt.Printf("DEBUG patterns: %v\n", patterns)
	
	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			fmt.Printf("DEBUG: Found pattern '%s' at %d\n", pattern, idx)
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

func main() {
	test3 := `我发现目前只有一个任务"写歌任务"，而您提到的"前两个任务"可能包括之前创建的"打谱"任务。让我先创建您要求的新任务，依赖关系将基于现有任务：<|tool_calls_begin |><|tool_call_begin | >create_task< | tool_sep | >{"creator_id":"@@73b809f17e9df57e9d2ed49289ada045245d96776c742648e7e547acb29c211c", "title": "系统测试依赖关系","content": "任务系统测试依赖关系。截止日期今天6点。依赖于前两个任务。", "due_time": "今天 18:00","dependencies": [1]}< |tool_call_end |><|tool_calls_end|>`
	
	beginMarker := "<|tool_calls_begin|>"
	endMarker := "<|tool_calls_end|>"
	callBeginMarker := "<|tool_call_begin|>"
	callEndMarker := "<|tool_call_end|>"
	sepMarker := "<|tool_sep|>"
	
	beginIdx := findMarker(test3, beginMarker)
	endIdx := findMarker(test3, endMarker)
	
	fmt.Printf("beginIdx=%d, endIdx=%d\n", beginIdx, endIdx)
	
	// 找到开始标记的实际结束位置
	actualBeginEndIdx := beginIdx
	for actualBeginEndIdx < len(test3) && actualBeginEndIdx < endIdx {
		if test3[actualBeginEndIdx] == '>' {
			actualBeginEndIdx++
			break
		}
		actualBeginEndIdx++
	}
	
	// 找到结束标记的实际结束位置
	actualEndIdx := endIdx
	for actualEndIdx < len(test3) {
		if test3[actualEndIdx] == '>' {
			actualEndIdx++
			break
		}
		actualEndIdx++
		if actualEndIdx >= len(test3) {
			break
		}
	}
	
	toolCallsBlock := test3[actualBeginEndIdx:endIdx]
	fmt.Printf("toolCallsBlock: %s\n", toolCallsBlock)
	fmt.Printf("toolCallsBlock length: %d\n", len(toolCallsBlock))
	
	// 尝试在 toolCallsBlock 中查找 callBeginMarker
	callBeginIdx := findMarker(toolCallsBlock, callBeginMarker)
	fmt.Printf("callBeginIdx in toolCallsBlock: %d\n", callBeginIdx)
	
	if callBeginIdx != -1 {
		// 找到开始标记的实际结束位置
		callBeginEndIdx := callBeginIdx
		for callBeginEndIdx < len(toolCallsBlock) {
			if toolCallsBlock[callBeginEndIdx] == '>' {
				callBeginEndIdx++
				break
			}
			callBeginEndIdx++
		}
		
		remaining := toolCallsBlock[callBeginEndIdx:]
		fmt.Printf("remaining after callBegin: %s\n", remaining[:min(100, len(remaining))])
		
		callEndIdx := findMarker(remaining, callEndMarker)
		fmt.Printf("callEndIdx in remaining: %d\n", callEndIdx)
		
		if callEndIdx != -1 {
			callContent := remaining[:callEndIdx]
			fmt.Printf("callContent: %s\n", callContent)
			
			sepIdx := findMarker(callContent, sepMarker)
			fmt.Printf("sepIdx: %d\n", sepIdx)
			
			if sepIdx != -1 {
				// 找到分隔符的实际结束位置
				sepEndIdx := sepIdx
				for sepEndIdx < len(callContent) {
					if callContent[sepEndIdx] == '>' {
						sepEndIdx++
						break
					}
					sepEndIdx++
				}
				
				toolName := strings.TrimSpace(callContent[:sepIdx])
				argsStr := strings.TrimSpace(callContent[sepEndIdx:])
				
				fmt.Printf("toolName: %s\n", toolName)
				fmt.Printf("argsStr: %s\n", argsStr[:min(200, len(argsStr))])
				
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
					fmt.Printf("JSON parse error: %v\n", err)
				} else {
					fmt.Printf("Parsed args: %v\n", args)
				}
			}
		}
	}
}

