# wechatbot-gpt
基于 DeepSeek AI 的微信机器人项目，能够自动回复微信私聊和群聊消息。项目基于[openwechat](https://github.com/eatmoreapple/openwechat)开发。

### 目前实现了以下功能
 + 群聊@回复
 + 私聊回复
 + 自动通过好友申请
 + DeepSeek AI 对话
 + Agent 功能（工具调用）
 
# 获取 DeepSeek API 密钥
访问 [DeepSeek 官网](https://www.deepseek.com/) 注册账号并获取 API 密钥

# 安装使用

## 1. 获取项目
```bash
git clone https://github.com/869413421/wechatbot.git
cd wechatbot
```

## 2. 配置
创建 `config.json` 文件：
```json
{
  "api_key": "your-deepseek-api-key",
  "model_name": "deepseek-chat",
  "auto_pass": true,
  "max_msg": 15
}
```

**配置说明**：
- `api_key`: DeepSeek API 密钥（必需）
- `model_name`: 模型名称，可选值：`deepseek-chat`（默认）、`deepseek-coder`
- `auto_pass`: 是否自动通过好友申请
- `max_msg`: 会话最大消息数

## 3. 启动
```bash
go run main.go
```

启动后会显示二维码，使用微信扫码登录即可。
