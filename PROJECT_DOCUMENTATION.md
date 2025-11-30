# 微信机器人 GPT 项目文档

## 一、项目概述

### 1.1 项目简介

这是一个基于 Go 语言开发的微信机器人项目，集成了 OpenAI GPT API，能够自动回复微信私聊和群聊消息。项目采用模块化设计，支持会话管理、角色定制、上下文记忆等功能。

### 1.2 核心功能

- ✅ **私聊自动回复**: 自动回复好友的文本消息
- ✅ **群聊@回复**: 在群聊中被@时自动回复
- ✅ **好友自动通过**: 可配置自动通过好友申请
- ✅ **GPT 对话**: 集成 OpenAI GPT-3.5-turbo 模型
- ✅ **会话管理**: 为每个用户/群组维护独立的对话上下文
- ✅ **角色定制**: 支持通过命令自定义机器人角色
- ✅ **对话历史**: 支持查看和管理对话历史

### 1.3 技术栈

- **语言**: Go 1.16+
- **微信 SDK**: openwechat v1.4.10
- **AI 模型**: OpenAI GPT-3.5-turbo
- **架构**: 模块化分层架构

---

## 二、项目架构

### 2.1 目录结构

```
wechatbot-gpt/
├── main.go                    # 程序入口
├── app/                       # 应用层 - 业务逻辑模块
│   ├── config/               # 配置模块
│   │   ├── models.go         # 配置模型
│   │   ├── methods.go        # 配置加载方法
│   │   └── enums.go          # 配置常量
│   ├── message/              # 消息处理模块
│   │   ├── models.go         # 消息处理器模型
│   │   ├── methods.go        # 消息处理方法
│   │   └── enums.go          # 消息类型枚举
│   └── gpt/                  # GPT 服务模块
│       ├── models.go         # GPT 请求/响应模型
│       ├── methods.go        # GPT API 调用方法
│       └── enums.go          # GPT 相关常量
├── components/               # 组件层 - 可复用组件
│   └── bootstrap/            # 启动组件
│       └── bootstrap.go      # 启动逻辑
├── config.dev.json           # 配置文件模板
└── go.mod                    # Go 模块依赖
```

### 2.2 架构分层

```
┌─────────────────────────────────────┐
│         main.go (入口层)             │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   components/bootstrap (启动层)      │
│   - 初始化微信机器人                  │
│   - 注册消息处理器                    │
│   - 处理登录逻辑                      │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   app/message (消息处理层)           │
│   - 消息路由                          │
│   - 私聊/群聊处理                     │
│   - 好友申请处理                      │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   app/gpt (AI 服务层)                │
│   - GPT API 调用                     │
│   - 会话管理                          │
│   - 上下文维护                        │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│   app/config (配置层)                │
│   - 配置加载                          │
│   - 环境变量支持                      │
└─────────────────────────────────────┘
```

### 2.3 模块依赖关系

```
main.go
  └── components/bootstrap
        ├── app/message
        │     ├── app/config
        │     └── app/gpt
        │           └── app/config
        └── openwechat (第三方)
```

---

## 三、核心模块详解

### 3.1 配置模块 (app/config)

**职责**: 管理项目配置，支持文件配置和环境变量

**关键组件**:
- `Configuration`: 配置结构体
  - `ApiKey`: OpenAI API 密钥
  - `AutoPass`: 是否自动通过好友申请
  - `MaxMsg`: 最大消息数量（用于会话管理）

**加载流程**:
1. 从 `config.json` 文件读取配置
2. 检查环境变量，环境变量优先级更高
3. 使用 `sync.Once` 确保只加载一次（单例模式）

**文件说明**:
- `models.go`: `Configuration` 结构体定义
- `methods.go`: `LoadConfig()` 配置加载逻辑
- `enums.go`: `HelpText` 帮助文本常量

---

### 3.2 消息处理模块 (app/message)

**职责**: 处理微信消息，路由到对应的处理器

**核心组件**:

1. **MessageHandlerInterface**: 消息处理器接口
   ```go
   type MessageHandlerInterface interface {
       handle(*openwechat.Message) error
       ReplyText(*openwechat.Message) error
   }
   ```

2. **UserMessageHandler**: 私聊消息处理器
   - 处理所有私聊文本消息
   - 调用 GPT API 生成回复
   - 维护用户会话 ID

3. **GroupMessageHandler**: 群聊消息处理器
   - 只处理@机器人的消息
   - 提取@文本并调用 GPT
   - 回复时@原发送者

**消息路由逻辑**:
```
收到消息
  ├── 是群消息? 
  │     └── 是 → GroupMessageHandler
  │
  ├── 是好友申请?
  │     └── 是 → 根据配置决定是否自动通过
  │
  └── 其他 → UserMessageHandler (私聊)
```

**特殊命令处理**:
- `help`: 返回帮助文本
- `role:xxx`: 设置机器人角色
- `get:role`: 获取当前角色
- `get:session`: 获取对话历史
- `换个话题` / `清空对话`: 清空对话历史

**文件说明**:
- `models.go`: 接口和结构体定义
- `methods.go`: 消息处理逻辑
- `enums.go`: 处理器类型枚举

---

### 3.3 GPT 服务模块 (app/gpt)

**职责**: 与 OpenAI API 交互，管理对话会话

**核心功能**:

1. **会话管理** (`sessionMap`)
   - 每个用户/群组有独立的会话 ID
   - 会话 ID 格式: `{NickName}-{UserName}`
   - 维护对话历史，支持上下文记忆

2. **消息结构**
   ```go
   type Message struct {
       Role    string // "system", "user", "assistant"
       Content string
   }
   ```

3. **会话限制**
   - 默认系统消息: "你是一个非常有帮助的聊天机器人"
   - 最大消息数: `MaxMsg * 2 + 1` (配置项)
   - 超出限制时删除最早的对话（保留 system 消息）

**API 调用流程**:
```
1. 构建请求体 (ChatGPTRequestBody)
   ├── Model: "gpt-3.5-turbo"
   └── Messages: 会话历史（包含 system, user, assistant）

2. 发送 HTTP POST 请求
   ├── URL: https://api.openai.com/v1/chat/completions
   ├── Headers: Authorization, Content-Type
   └── Body: JSON 格式的请求体

3. 解析响应 (ChatGPTResponseBody)
   └── 提取 assistant 回复内容

4. 更新会话
   └── 将用户消息和 AI 回复添加到会话历史
```

**特殊命令处理**:
- `role:xxx`: 更新 system 消息（角色定义）
- `换个话题`: 清空对话历史（保留 system）
- `get:role`: 返回当前 system 消息
- `get:session`: 返回格式化后的对话历史

**文件说明**:
- `models.go`: GPT 请求/响应结构体
- `methods.go`: API 调用和会话管理逻辑
- `enums.go`: API 端点常量

---

### 3.4 启动组件 (components/bootstrap)

**职责**: 初始化微信机器人，处理登录和消息注册

**启动流程**:
1. 创建微信机器人实例（桌面模式）
2. 注册消息处理函数 (`message.Handler`)
3. 注册二维码回调（用于登录）
4. 尝试热登录（使用 `storage.json`）
5. 如果热登录失败，执行普通登录
6. 阻塞主 goroutine，保持运行

**关键配置**:
- 登录模式: `openwechat.Desktop` (桌面模式)
- 存储方式: JSON 文件热重载 (`storage.json`)

---

## 四、时间流程

### 4.1 应用启动流程

```
[1] main() 函数启动
    │
    └──> [2] bootstrap.Run()
            │
            ├──> [3] 创建微信机器人实例
            │       └──> openwechat.DefaultBot(Desktop)
            │
            ├──> [4] 注册消息处理器
            │       └──> bot.MessageHandler = message.Handler
            │
            ├──> [5] 注册二维码回调
            │       └──> bot.UUIDCallback = PrintlnQrcodeUrl
            │
            ├──> [6] 尝试热登录
            │       ├──> 成功 → [8]
            │       └──> 失败 → [7]
            │
            ├──> [7] 执行普通登录
            │       └──> 显示二维码，等待扫码
            │
            └──> [8] bot.Block() 阻塞运行
                    └──> 等待消息...
```

### 4.2 消息处理流程

```
[1] 微信收到消息
    │
    └──> [2] openwechat SDK 触发回调
            │
            └──> [3] message.Handler() 被调用
                    │
                    ├──> [4] 判断消息类型
                    │       │
                    │       ├──> 群消息?
                    │       │       └──> [5a] GroupMessageHandler.handle()
                    │       │
                    │       ├──> 好友申请?
                    │       │       └──> [5b] 根据配置决定是否自动通过
                    │       │
                    │       └──> 其他
                    │               └──> [5c] UserMessageHandler.handle()
                    │
                    └──> [6] 处理消息
                            │
                            ├──> [7] 提取消息内容
                            │       └──> 去除@文本（群聊）
                            │
                            ├──> [8] 判断特殊命令
                            │       ├──> help → 返回帮助文本
                            │       ├──> role:xxx → 设置角色
                            │       ├──> get:role → 获取角色
                            │       ├──> get:session → 获取历史
                            │       ├──> 换个话题 → 清空历史
                            │       └──> 普通消息 → [9]
                            │
                            ├──> [9] 调用 gpt.Completions()
                            │       │
                            │       ├──> [10] 获取/创建会话
                            │       │       └──> sessionId = {NickName}-{UserName}
                            │       │
                            │       ├──> [11] 添加用户消息到会话
                            │       │
                            │       ├──> [12] 构建 GPT 请求
                            │       │       └──> 包含会话历史
                            │       │
                            │       ├──> [13] 发送 HTTP 请求到 OpenAI
                            │       │
                            │       ├──> [14] 解析响应，获取回复
                            │       │
                            │       └──> [15] 添加 AI 回复到会话
                            │
                            └──> [16] 回复消息到微信
                                    ├──> 私聊: 直接回复
                                    └──> 群聊: @原发送者后回复
```

### 4.3 会话管理流程

```
[1] 收到消息，生成 sessionId
    │
    └──> [2] 检查 sessionMap[sessionId] 是否存在
            │
            ├──> 不存在 → [3] 创建新会话
            │       └──> 添加默认 system 消息
            │
            └──> 存在 → [4] 获取现有会话
                    │
                    └──> [5] 添加用户消息
                            │
                            └──> [6] 检查消息数量
                                    │
                                    ├──> 超过限制 → [7] 删除最早消息（保留 system）
                                    └──> 未超过 → [8] 正常添加
```

---

## 五、数据流

### 5.1 消息数据流

```
微信客户端
    │
    │ (消息事件)
    ▼
openwechat SDK
    │
    │ (*openwechat.Message)
    ▼
message.Handler()
    │
    │ (路由判断)
    ├──> GroupMessageHandler
    │       │
    │       │ (提取消息内容)
    │       ▼
    │   requestText (去除@文本)
    │       │
    │       │ (生成 sessionId)
    │       ▼
    │   sessionId = "群名-群ID"
    │
    └──> UserMessageHandler
            │
            │ (提取消息内容)
            ▼
        requestText
            │
            │ (生成 sessionId)
            ▼
        sessionId = "用户名-用户ID"
            │
            │ (统一流程)
            ▼
        gpt.Completions(sessionId, requestText, "")
            │
            │ (获取会话历史)
            ▼
        sessionMap[sessionId] → []Message
            │
            │ (构建请求)
            ▼
        HTTP POST → OpenAI API
            │
            │ (解析响应)
            ▼
        reply (AI 回复)
            │
            │ (更新会话)
            ▼
        sessionMap[sessionId] += [用户消息, AI回复]
            │
            │ (返回回复)
            ▼
        msg.ReplyText(reply)
            │
            ▼
        微信客户端 (用户收到回复)
```

### 5.2 会话数据结构

```go
// 会话存储结构
sessionMap = map[string][]Message

// 示例会话数据
sessionMap["张三-wxid_123"] = []Message{
    {Role: "system", Content: "你是一个非常有帮助的聊天机器人"},
    {Role: "user", Content: "你好"},
    {Role: "assistant", Content: "你好！有什么我可以帮助你的吗？"},
    {Role: "user", Content: "今天天气怎么样？"},
    {Role: "assistant", Content: "我无法获取实时天气信息..."},
    // ... 更多消息
}
```

---

## 六、关键逻辑说明

### 6.1 会话 ID 生成规则

**私聊**:
```go
sessionId = user.NickName + "-" + user.UserName
// 示例: "张三-wxid_abc123"
```

**群聊**:
```go
sessionId = group.NickName + "-" + group.UserName
// 示例: "技术交流群-wxid_group123"
```

**特点**:
- 每个用户/群组有独立会话
- 基于用户名和 ID 组合，确保唯一性
- 支持多用户同时对话，互不干扰

### 6.2 消息数量限制机制

```go
// 配置: MaxMsg = 15
// 限制: MaxMsg * 2 + 1 = 31 条消息

// 当会话超过 31 条消息时:
// 1. 保留第一条 system 消息
// 2. 删除最早的 user + assistant 消息对
// 3. 保持会话在合理范围内
```

**目的**:
- 控制 API 调用成本（token 数量）
- 保持上下文相关性（最近对话更重要）
- 避免会话过长导致性能问题

### 6.3 角色定制机制

**设置角色**:
```
用户发送: role:你是一个专业的Python编程助手
```

**处理流程**:
1. 正则匹配 `role:(.*)`
2. 提取角色描述
3. 更新会话的 system 消息
4. 后续对话将基于新角色进行

**获取角色**:
```
用户发送: get:role
返回: role: 你是一个专业的Python编程助手
```

### 6.4 群聊@处理机制

**触发条件**:
- 消息来自群聊 (`msg.IsSendByGroup()`)
- 消息@了机器人 (`msg.IsAt()`)

**处理流程**:
1. 获取发送者信息
2. 提取机器人昵称 (`sender.Self().NickName`)
3. 从消息内容中移除 `@机器人昵称`
4. 调用 GPT 生成回复
5. 回复时@原发送者 (`@发送者昵称\n回复内容`)

**示例**:
```
原始消息: "@小助手 今天天气怎么样？"
处理后: "今天天气怎么样？"
回复: "@张三\n我无法获取实时天气信息..."
```

---

## 七、配置说明

### 7.1 配置文件 (config.json)

```json
{
  "api_key": "your-openai-api-key",
  "auto_pass": true,
  "max_msg": 15
}
```

**配置项说明**:
- `api_key`: OpenAI API 密钥（必需）
- `auto_pass`: 是否自动通过好友申请（默认: false）
- `max_msg`: 会话最大消息数（默认: 15）

### 7.2 环境变量支持

支持通过环境变量覆盖配置:
- `ApiKey`: 覆盖 `api_key`
- `AutoPass`: 覆盖 `auto_pass` (值为 "true" 时启用)
- `MaxMsg`: 覆盖 `max_msg`

**优先级**: 环境变量 > 配置文件

---

## 八、命令列表

### 8.1 用户可用命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `help` | 显示帮助信息 | `help` |
| `role:xxx` | 设置机器人角色 | `role:你是一个专业的程序员` |
| `get:role` | 获取当前角色 | `get:role` |
| `get:session` | 获取对话历史 | `get:session` |
| `换个话题` | 清空对话历史 | `换个话题` |
| `清空对话` | 清空对话历史 | `清空对话` |

### 8.2 帮助文本

```
欢迎使用聊天机器人，你可以通过以下命令来使用聊天机器人：
1. 输入"help"查看帮助
2. 输入"role:" 定义角色行为 格式: "role:<你想定制的机器人角色行为>"
	-- 例如: role:你是一个非常有帮助的聊天机器人(这个是默认的角色行为)
3. 输入"get:role" 获取当前角色行为
4. 输入"get:session" 获取当前和你话题的聊天记录
5. 输入"换个话题" 重新开始一个话题(会清空当前和你话题的聊天记录保留角色行为)
---------------
更多功能正在开发中，敬请期待
```

---

## 九、错误处理

### 9.1 错误处理机制

**GPT API 调用失败**:
```go
if err != nil {
    log.Printf("gtp request error: %v \n", err)
    msg.ReplyText("机器人神了，我一会发现了就去修。")
    return err
}
```

**消息发送失败**:
```go
if err != nil {
    log.Printf("response user error: %v \n", err)
    return err
}
```

**配置加载失败**:
```go
if err != nil {
    log.Fatalf("open config err: %v", err)
    return
}
```

### 9.2 日志记录

项目使用标准 `log` 包记录日志:
- 消息接收日志
- GPT 请求/响应日志
- 错误日志
- 调试信息

---

## 十、扩展建议

### 10.1 功能扩展方向

1. **多模型支持**: 支持其他 AI 模型（Claude、Gemini 等）
2. **图片处理**: 支持图片消息的识别和处理
3. **文件处理**: 支持文件上传和下载
4. **定时任务**: 支持定时发送消息
5. **数据持久化**: 将会话数据保存到数据库
6. **插件系统**: 支持插件扩展功能
7. **Web 管理界面**: 提供 Web 界面管理配置和查看日志

### 10.2 性能优化

1. **并发处理**: 使用 goroutine 并发处理多个消息
2. **缓存机制**: 缓存常用配置和会话数据
3. **连接池**: 复用 HTTP 连接
4. **限流控制**: 限制 API 调用频率

### 10.3 安全性增强

1. **API 密钥加密**: 加密存储 API 密钥
2. **访问控制**: 支持白名单/黑名单
3. **消息过滤**: 过滤敏感内容
4. **日志脱敏**: 敏感信息脱敏处理

---

## 十一、常见问题

### Q1: 如何更换 OpenAI API 密钥？

**A**: 修改 `config.json` 中的 `api_key` 字段，或设置环境变量 `ApiKey`。

### Q2: 如何清空所有会话？

**A**: 重启程序即可，会话数据存储在内存中（`sessionMap`）。

### Q3: 群聊中机器人不回复？

**A**: 确保消息@了机器人，且机器人昵称配置正确。

### Q4: 如何查看对话历史？

**A**: 发送 `get:session` 命令即可查看当前会话的对话历史。

### Q5: 如何自定义机器人角色？

**A**: 发送 `role:你的角色描述` 命令，例如 `role:你是一个专业的Python编程助手`。

---

## 十二、开发指南

### 12.1 本地开发

```bash
# 1. 克隆项目
git clone <repository-url>
cd wechatbot-gpt

# 2. 复制配置文件
copy config.dev.json config.json

# 3. 编辑配置文件
# 修改 config.json 中的 api_key

# 4. 安装依赖
go mod download

# 5. 运行项目
go run main.go
```

### 12.2 构建部署

```bash
# 构建可执行文件
go build -o wechatbot.exe

# 运行
./wechatbot.exe
```

### 12.3 代码规范

- 遵循 Go 代码规范
- 使用有意义的变量和函数名
- 添加必要的注释
- 错误处理要完善

---

## 十三、版本信息

- **当前版本**: v1.0.0
- **Go 版本**: 1.16+
- **openwechat 版本**: v1.4.10
- **最后更新**: 2024年

---

**文档维护**: 请及时更新本文档以反映项目的最新状态。

