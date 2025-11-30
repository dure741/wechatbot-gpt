# DeepSeek Agent 功能说明

## 功能概述

为 DeepSeek 模型添加了 Agent 功能，允许模型调用本地工具执行操作。当前支持的功能是：**打开终端并输出文本**。

## 工作原理

### 1. Function Calling 机制

DeepSeek 支持类似 OpenAI 的 function calling 机制：
- 在请求中提供可用工具列表
- 模型可以选择调用工具
- 执行工具后，将结果返回给模型
- 模型基于工具结果生成最终回复

### 2. 工具执行流程

```
用户消息
  ↓
DeepSeek API (带工具定义)
  ↓
模型决定调用工具
  ↓
执行工具 (如：打开终端)
  ↓
将工具结果返回给模型
  ↓
模型生成最终回复
```

## 当前支持的工具

### open_terminal_output

**功能**: 打开电脑终端并输出指定文本

**参数**:
- `text` (string, 可选): 要在终端中输出的文本，默认为 "agent done"

**使用示例**:
- 用户: "帮我打开终端输出 agent done"
- DeepSeek 会调用 `open_terminal_output` 工具
- 系统会打开新终端窗口并输出 "agent done"

## 技术实现

### 1. Agent 执行器 (`app/agent/executor.go`)

负责执行具体的工具命令：

```go
type Executor struct {
}

func (e *Executor) ExecuteCommand(command string, args map[string]interface{}) (string, error)
```

**支持的操作系统**:
- **Windows**: 使用 PowerShell 打开新窗口
- **macOS**: 使用 Terminal.app
- **Linux**: 使用 gnome-terminal 或 xterm

### 2. DeepSeek Provider 增强 (`app/llm/deepseek.go`)

- 在请求中包含工具定义
- 检测模型返回的工具调用
- 执行工具并获取结果
- 将结果返回给模型获取最终回复

## 使用方法

### 1. 配置要求

确保 `config.json` 中配置了 DeepSeek：

```json
{
  "model_type": "deepseek",
  "deepseek_api_key": "your-deepseek-api-key",
  "model_name": "deepseek-chat"
}
```

### 2. 使用示例

**示例 1: 打开终端输出默认文本**
```
用户: 打开终端输出 agent done
```

**示例 2: 打开终端输出自定义文本**
```
用户: 打开终端并输出 "Hello World"
```

**示例 3: 自然语言请求**
```
用户: 帮我打开一个终端窗口，显示 agent done
```

## 代码结构

```
app/
├── agent/              # Agent 功能模块
│   └── executor.go    # 工具执行器
└── llm/
    └── deepseek.go    # DeepSeek Provider (支持工具调用)
```

## 扩展新工具

要添加新的工具，需要：

1. **在 `executor.go` 中添加工具执行逻辑**:

```go
func (e *Executor) ExecuteCommand(command string, args map[string]interface{}) (string, error) {
    switch command {
    case "open_terminal_output":
        return e.openTerminalOutput(args)
    case "your_new_command":  // 新工具
        return e.yourNewCommand(args)
    default:
        return "", fmt.Errorf("unknown command: %s", command)
    }
}
```

2. **在 `GetAvailableCommands` 中添加工具定义**:

```go
func (e *Executor) GetAvailableCommands() []map[string]interface{} {
    return []map[string]interface{}{
        {
            "name":        "open_terminal_output",
            "description": "...",
            "parameters":  {...},
        },
        {
            "name":        "your_new_command",  // 新工具定义
            "description": "工具描述",
            "parameters":  {...},
        },
    }
}
```

## 注意事项

1. **安全性**: 
   - 工具执行具有系统权限，需要谨慎设计
   - 建议添加权限控制和命令白名单

2. **错误处理**:
   - 工具执行失败会记录日志
   - 错误信息会返回给模型，模型可以基于错误信息调整策略

3. **跨平台兼容**:
   - 当前实现支持 Windows、macOS、Linux
   - 不同平台的命令可能不同

4. **API 兼容性**:
   - 需要确保 DeepSeek API 支持 function calling
   - 如果 API 不支持，工具调用会被忽略

## 故障排查

### 问题 1: 工具调用不生效

**可能原因**:
- DeepSeek API 不支持 function calling
- 工具定义格式不正确
- 模型未识别需要调用工具

**解决方案**:
- 检查 API 响应中的 `tool_calls` 字段
- 查看日志确认工具定义是否正确发送
- 尝试更明确的用户指令

### 问题 2: 终端无法打开

**可能原因**:
- 操作系统不支持
- 缺少必要的命令行工具
- 权限问题

**解决方案**:
- 检查操作系统类型
- 确保安装了 PowerShell (Windows) 或 Terminal (macOS/Linux)
- 检查执行权限

### 问题 3: 工具执行但无响应

**可能原因**:
- 工具执行成功但模型未生成回复
- API 调用链中断

**解决方案**:
- 查看日志确认工具执行结果
- 检查第二次 API 调用是否成功

## 日志示例

**正常工具调用流程**:
```
request DeepSeek deepseek-chat json string : {...tools...}
DeepSeek requested tool calls: 1
Agent executing command: open_terminal_output with args: map[text:agent done]
Opening terminal and outputting: agent done
Successfully opened terminal with output: agent done
request DeepSeek deepseek-chat json string : {...tool results...}
DeepSeek response text: 已成功打开终端并输出 "agent done"
```

## 未来扩展

可以添加的工具示例：
- 文件操作（读取、写入、删除）
- 网络请求
- 数据库查询
- 系统信息获取
- 定时任务
- 等等

---

**最后更新**: 2024年
**功能版本**: v1.0

