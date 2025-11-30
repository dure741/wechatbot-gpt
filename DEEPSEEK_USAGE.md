# DeepSeek 模型使用指南

## 一、概述

项目现已支持 DeepSeek 模型，可以在 OpenAI 和 DeepSeek 之间切换使用。DeepSeek 提供了兼容 OpenAI API 格式的接口，使用方式与 OpenAI 类似。

## 二、配置说明

### 2.1 配置文件 (config.json)

在 `config.json` 中添加以下配置项：

```json
{
  "api_key": "your-openai-api-key",
  "deepseek_api_key": "your-deepseek-api-key",
  "model_type": "deepseek",
  "model_name": "deepseek-chat",
  "auto_pass": true,
  "max_msg": 15
}
```

### 2.2 配置项说明

| 配置项 | 类型 | 说明 | 默认值 |
|--------|------|------|--------|
| `api_key` | string | OpenAI API 密钥 | 必需 |
| `deepseek_api_key` | string | DeepSeek API 密钥 | 可选（如果使用 DeepSeek） |
| `model_type` | string | 模型类型：`"openai"` 或 `"deepseek"` | `"openai"` |
| `model_name` | string | 模型名称 | `"gpt-3.5-turbo"` (OpenAI) 或 `"deepseek-chat"` (DeepSeek) |
| `auto_pass` | bool | 是否自动通过好友申请 | `false` |
| `max_msg` | int | 会话最大消息数 | `15` |

### 2.3 使用 OpenAI

```json
{
  "api_key": "sk-...",
  "model_type": "openai",
  "model_name": "gpt-3.5-turbo",
  "auto_pass": true,
  "max_msg": 15
}
```

### 2.4 使用 DeepSeek

```json
{
  "api_key": "sk-...",
  "deepseek_api_key": "sk-...",
  "model_type": "deepseek",
  "model_name": "deepseek-chat",
  "auto_pass": true,
  "max_msg": 15
}
```

**注意**: 如果 `deepseek_api_key` 未配置，系统会使用 `api_key` 作为 DeepSeek 的 API 密钥。

## 三、环境变量配置

也可以通过环境变量进行配置：

```bash
# Windows PowerShell
$env:ModelType = "deepseek"
$env:DeepSeekApiKey = "sk-..."
$env:ModelName = "deepseek-chat"

# Linux/Mac
export ModelType=deepseek
export DeepSeekApiKey=sk-...
export ModelName=deepseek-chat
```

**优先级**: 环境变量 > 配置文件

## 四、DeepSeek 模型说明

### 4.1 支持的模型

DeepSeek 提供以下模型（根据 DeepSeek API 文档）：

- `deepseek-chat`: 通用对话模型（推荐）
- `deepseek-coder`: 代码专用模型

### 4.2 API 端点

- **DeepSeek API**: `https://api.deepseek.com/v1/chat/completions`
- **OpenAI API**: `https://api.openai.com/v1/chat/completions`

### 4.3 API 兼容性

DeepSeek API 完全兼容 OpenAI API 格式，包括：
- 请求格式相同
- 响应格式相同
- 认证方式相同（Bearer Token）

## 五、使用示例

### 5.1 切换到 DeepSeek

1. **编辑配置文件** `config.json`:
```json
{
  "model_type": "deepseek",
  "deepseek_api_key": "your-deepseek-api-key",
  "model_name": "deepseek-chat"
}
```

2. **重启程序**:
```bash
go run main.go
```

3. **测试**: 发送消息给机器人，应该会使用 DeepSeek 模型回复。

### 5.2 切换回 OpenAI

1. **编辑配置文件** `config.json`:
```json
{
  "model_type": "openai",
  "api_key": "your-openai-api-key",
  "model_name": "gpt-3.5-turbo"
}
```

2. **重启程序**

## 六、获取 DeepSeek API 密钥

1. 访问 DeepSeek 官网: https://www.deepseek.com/
2. 注册/登录账号
3. 进入 API 管理页面
4. 创建 API 密钥
5. 将密钥配置到 `config.json` 的 `deepseek_api_key` 字段

## 七、功能特性

### 7.1 已支持的功能

- ✅ 自动根据配置选择模型
- ✅ 支持 OpenAI 和 DeepSeek 切换
- ✅ 会话管理（两个模型独立管理）
- ✅ 角色定制
- ✅ 对话历史查看
- ✅ 所有原有功能

### 7.2 会话隔离

不同模型类型的会话是独立的：
- OpenAI 会话: 使用 OpenAI 的会话历史
- DeepSeek 会话: 使用 DeepSeek 的会话历史

切换模型类型时，会话历史会重新开始。

## 八、常见问题

### Q1: 如何知道当前使用的是哪个模型？

**A**: 查看日志输出，会显示使用的模型名称，例如：
```
request deepseek-chat json string : {...}
```

### Q2: 可以同时使用两个模型吗？

**A**: 不可以，一次只能使用一个模型。但可以通过修改配置并重启程序来切换。

### Q3: DeepSeek API 密钥在哪里获取？

**A**: 访问 DeepSeek 官网，登录后在 API 管理页面创建密钥。

### Q4: DeepSeek 和 OpenAI 的响应速度有区别吗？

**A**: 响应速度取决于各自的 API 服务状态，通常差异不大。

### Q5: 切换模型后，之前的对话历史会保留吗？

**A**: 不会。切换模型类型后，会话历史会重新开始，因为不同模型的会话是独立管理的。

## 九、技术实现

### 9.1 代码变更

1. **配置模块** (`app/config/`):
   - 添加 `ModelType`、`ModelName`、`DeepSeekApiKey` 字段
   - 支持环境变量配置

2. **GPT 模块** (`app/gpt/`):
   - 添加 `getApiConfig()` 函数，根据配置返回 API URL 和密钥
   - 修改 `Completions()` 函数，支持动态选择模型

3. **API 端点**:
   - OpenAI: `https://api.openai.com/v1/chat/completions`
   - DeepSeek: `https://api.deepseek.com/v1/chat/completions`

### 9.2 架构设计

```
配置加载
  │
  ├──> model_type = "openai" → 使用 OpenAI API
  │       ├──> baseURL = OpenAIBaseURL
  │       ├──> apiKey = config.ApiKey
  │       └──> modelName = config.ModelName (默认: gpt-3.5-turbo)
  │
  └──> model_type = "deepseek" → 使用 DeepSeek API
          ├──> baseURL = DeepSeekBaseURL
          ├──> apiKey = config.DeepSeekApiKey (或 config.ApiKey)
          └──> modelName = config.ModelName (默认: deepseek-chat)
```

## 十、注意事项

1. **API 密钥安全**: 不要将包含 API 密钥的 `config.json` 提交到版本控制系统
2. **模型切换**: 切换模型需要重启程序才能生效
3. **会话隔离**: 不同模型的会话历史是独立的
4. **API 限制**: 注意各自的 API 调用频率限制
5. **成本考虑**: 不同模型的定价可能不同，请查看各自的定价页面

---

**最后更新**: 2024年
**文档版本**: v1.0

