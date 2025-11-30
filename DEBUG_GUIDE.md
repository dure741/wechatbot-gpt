# Cursor 中调试 Go 代码指南

## 一、前置准备

### 1.1 安装 Go 扩展

Cursor 基于 VS Code，需要安装 Go 扩展：

1. 打开扩展面板：`Ctrl+Shift+X` (Windows) 或 `Cmd+Shift+X` (Mac)
2. 搜索 "Go"
3. 安装官方扩展：**Go** (由 Go Team at Google 提供)

### 1.2 安装 Delve 调试器

Go 调试需要 Delve 调试器：

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 验证安装
dlv version
```

**注意**: 确保 `$GOPATH/bin` 或 `$GOBIN` 在系统 PATH 中。

## 二、调试配置

### 2.1 配置文件位置

项目已创建 `.vscode/launch.json` 配置文件，包含以下调试配置：

1. **Launch Program**: 基本启动配置
2. **Launch with Config**: 带环境变量的启动配置
3. **Attach to Process**: 附加到运行中的进程
4. **Debug Test**: 调试测试用例

### 2.2 配置说明

**Launch Program**:
```json
{
    "name": "Launch Program",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}",
    "env": {},
    "args": []
}
```

**Launch with Config** (推荐用于本项目):
```json
{
    "name": "Launch with Config",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "${workspaceFolder}",
    "env": {
        "ApiKey": "${env:ApiKey}",
        "ModelType": "${env:ModelType}",
        "DeepSeekApiKey": "${env:DeepSeekApiKey}"
    }
}
```

## 三、使用调试功能

### 3.1 设置断点

1. **在代码行号左侧点击**，会出现红色圆点
2. **条件断点**: 右键点击断点，可以设置条件
3. **日志断点**: 右键点击断点，选择 "Add Logpoint"

**常用断点位置**:
- `main.go:8` - 程序入口
- `app/message/methods.go:23` - 消息处理入口
- `app/session/methods.go:13` - 会话处理入口
- `app/llm/openai.go` 或 `app/llm/deepseek.go` - AI 调用

### 3.2 启动调试

**方法一：使用快捷键**
- `F5` - 开始调试（使用当前选中的配置）
- `Ctrl+F5` - 运行而不调试

**方法二：使用调试面板**
1. 点击左侧活动栏的调试图标（或按 `Ctrl+Shift+D`）
2. 在顶部下拉菜单中选择调试配置
3. 点击绿色播放按钮

**方法三：使用命令面板**
1. 按 `Ctrl+Shift+P` 打开命令面板
2. 输入 "Debug: Start Debugging"
3. 选择配置

### 3.3 调试控制

调试启动后，可以使用以下控制：

| 操作 | 快捷键 | 说明 |
|------|--------|------|
| 继续 | `F5` | 继续执行到下一个断点 |
| 单步跳过 | `F10` | 执行当前行，不进入函数 |
| 单步进入 | `F11` | 进入函数内部 |
| 单步跳出 | `Shift+F11` | 跳出当前函数 |
| 重启 | `Ctrl+Shift+F5` | 重新启动调试 |
| 停止 | `Shift+F5` | 停止调试 |

### 3.4 查看变量

**变量面板**:
- 左侧调试面板的 "Variables" 区域
- 显示当前作用域的所有变量
- 可以展开查看结构体字段

**监视表达式**:
- 在 "Watch" 面板添加表达式
- 实时查看变量值

**调试控制台**:
- 在 "Debug Console" 中可以输入表达式
- 例如：输入变量名查看值
- 可以调用函数（需谨慎）

### 3.5 调用堆栈

在 "Call Stack" 面板可以查看：
- 当前执行的函数调用链
- 点击可以跳转到对应的代码位置
- 查看每个函数的局部变量

## 四、调试技巧

### 4.1 条件断点

**设置条件断点**:
1. 右键点击断点
2. 选择 "Edit Breakpoint"
3. 输入条件，例如：
   - `msg.Content == "help"`
   - `sessionId == "test-user"`
   - `len(messages) > 10`

### 4.2 日志断点

**设置日志断点**:
1. 右键点击断点
2. 选择 "Add Logpoint"
3. 输入日志表达式，例如：
   - `收到消息: {msg.Content}`
   - `会话ID: {sessionId}, 消息数: {len(messages)}`

### 4.3 调试特定功能

**调试消息处理**:
1. 在 `app/message/methods.go:23` 设置断点
2. 启动调试
3. 发送微信消息触发断点

**调试 AI 调用**:
1. 在 `app/llm/openai.go` 或 `app/llm/deepseek.go` 的 `Chat` 方法设置断点
2. 可以查看请求和响应数据

**调试会话管理**:
1. 在 `app/session/methods.go` 设置断点
2. 查看会话的创建和管理过程

### 4.4 环境变量配置

**方法一：在 launch.json 中配置**
```json
{
    "env": {
        "ApiKey": "your-api-key",
        "ModelType": "openai",
        "DeepSeekApiKey": "your-deepseek-key"
    }
}
```

**方法二：使用系统环境变量**
- Windows PowerShell:
  ```powershell
  $env:ApiKey = "your-api-key"
  $env:ModelType = "openai"
  ```
- 然后在 launch.json 中使用 `${env:ApiKey}`

**方法三：使用 .env 文件**
1. 安装 "dotenv" 扩展
2. 创建 `.env` 文件（不要提交到 Git）
3. 在 launch.json 中配置：
   ```json
   {
       "envFile": "${workspaceFolder}/.env"
   }
   ```

## 五、常见问题

### Q1: 调试时提示 "could not launch process"

**解决方案**:
1. 确保已安装 Delve: `go install github.com/go-delve/delve/cmd/dlv@latest`
2. 检查 Go 版本是否兼容
3. 尝试重新编译: `go build`

### Q2: 断点不生效

**解决方案**:
1. 确保代码已保存
2. 检查断点是否在可执行代码行（不是注释或空行）
3. 尝试重新启动调试
4. 检查 `showLog: true` 查看详细日志

### Q3: 无法查看变量值

**解决方案**:
1. 确保程序已暂停在断点处
2. 检查变量是否在当前作用域
3. 尝试在 Debug Console 中输入变量名

### Q4: 调试时程序运行很慢

**解决方案**:
1. 减少断点数量
2. 使用条件断点而不是普通断点
3. 使用日志断点代替普通断点

## 六、调试配置示例

### 6.1 调试消息处理流程

在 `app/message/methods.go` 的 `Handler` 函数设置断点：
```go
func Handler(msg *openwechat.Message) {
    // 在这里设置断点
    log.Printf("hadler Received msg : %v", msg.Content)
    // ...
}
```

### 6.2 调试 AI 调用

在 `app/llm/openai.go` 的 `Chat` 方法设置断点：
```go
func (p *OpenAIProvider) Chat(messages []Message) (string, error) {
    // 在这里设置断点，查看请求数据
    requestBody := map[string]interface{}{
        "model":    p.modelName,
        "messages": messages,
    }
    // ...
}
```

### 6.3 调试会话管理

在 `app/session/methods.go` 的 `Completions` 函数设置断点：
```go
func Completions(sessionId, msg string, change_str string) (string, error) {
    // 在这里设置断点，查看会话状态
    // ...
}
```

## 七、高级调试技巧

### 7.1 远程调试

如果需要调试运行在远程服务器上的程序：

```json
{
    "name": "Remote Debug",
    "type": "go",
    "request": "attach",
    "mode": "remote",
    "remotePath": "/path/to/remote/project",
    "port": 2345,
    "host": "127.0.0.1"
}
```

启动远程程序时使用：
```bash
dlv debug --headless --listen=:2345 --api-version=2
```

### 7.2 调试测试用例

使用 "Debug Test" 配置：
1. 在测试文件中设置断点
2. 选择 "Debug Test" 配置
3. 启动调试

### 7.3 使用调试控制台

在调试控制台中可以：
- 查看变量: 输入变量名
- 调用函数: `getSession("test-id")`
- 修改变量: `sessionId = "new-id"` (需谨慎)

## 八、快捷键总结

| 功能 | Windows/Linux | Mac |
|------|---------------|-----|
| 开始调试 | `F5` | `F5` |
| 运行不调试 | `Ctrl+F5` | `Cmd+F5` |
| 停止调试 | `Shift+F5` | `Shift+F5` |
| 重启调试 | `Ctrl+Shift+F5` | `Cmd+Shift+F5` |
| 单步跳过 | `F10` | `F10` |
| 单步进入 | `F11` | `F11` |
| 单步跳出 | `Shift+F11` | `Shift+F11` |
| 切换断点 | `F9` | `F9` |
| 打开调试面板 | `Ctrl+Shift+D` | `Cmd+Shift+D` |

---

**提示**: 
- 调试前确保 `config.json` 文件存在且配置正确
- 调试微信机器人时，需要先登录微信
- 建议在关键函数入口设置断点，逐步调试

