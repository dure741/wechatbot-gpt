package main

import (
	"fmt"
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
	
	fmt.Printf("DEBUG: baseMarker=%s, inner=%s, markerName=%s\n", baseMarker, inner, markerName)
	
	// 构建带空格的变体模式
	patterns := []string{
		"<|" + markerName + "|>",      // 标准格式
		"<|" + markerName + " |>",     // 在 | 和 > 之间有空格
		"<| " + markerName + "|>",    // 在 < 和 | 之间有空格
		"<| " + markerName + " |>",   // 两边都有空格
		"<|" + markerName + "| >",     // 在 | 和 > 之间有空格（另一种）
	}
	
	fmt.Printf("DEBUG: Trying patterns: %v\n", patterns)
	
	for _, pattern := range patterns {
		if idx := strings.Index(content, pattern); idx != -1 {
			fmt.Printf("DEBUG: Found pattern '%s' at index %d\n", pattern, idx)
			return idx
		}
	}
	
	return -1
}

func main() {
	test := `我发现目前只有一个任务"写歌任务"，而您提到的"前两个任务"可能包括之前创建的"打谱"任务。让我先创建您要求的新任务，依赖关系将基于现有任务：<|tool_calls_begin |><|tool_call_begin | >create_task< | tool_sep | >{"creator_id":"@@73b809f17e9df57e9d2ed49289ada045245d96776c742648e7e547acb29c211c", "title": "系统测试依赖关系","content": "任务系统测试依赖关系。截止日期今天6点。依赖于前两个任务。", "due_time": "今天 18:00","dependencies": [1]}< |tool_call_end |><|tool_calls_end|>`
	
	baseMarker := "<|tool_calls_begin|>"
	
	fmt.Printf("Looking for: %s\n", baseMarker)
	fmt.Printf("In content: ...%s...\n", test[200:300])
	
	idx := findMarker(test, baseMarker)
	fmt.Printf("Result: idx=%d\n", idx)
	
	if idx != -1 {
		fmt.Printf("Found at: ...%s...\n", test[max(0, idx-10):min(len(test), idx+50)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

