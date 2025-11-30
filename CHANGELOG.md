# 更新日志

## v2.0.0 - 单一模型支持

### 重大变更
- ✅ **移除 OpenAI 支持**：项目现在仅支持 DeepSeek 模型
- ✅ **简化配置**：移除了 `model_type`、`deepseek_api_key` 等冗余配置项
- ✅ **代码清理**：删除了 OpenAI provider 相关代码

### 配置变更

**旧配置** (v1.x):
```json
{
  "api_key": "your-openai-key",
  "deepseek_api_key": "your-deepseek-key",
  "model_type": "deepseek",
  "model_name": "deepseek-chat"
}
```

**新配置** (v2.0):
```json
{
  "api_key": "your-deepseek-api-key",
  "model_name": "deepseek-chat"
}
```

### 迁移指南

1. **更新配置文件**：
   - 将 `deepseek_api_key` 的值移到 `api_key`
   - 删除 `model_type` 字段
   - 保留 `model_name`（可选，默认为 "deepseek-chat"）

2. **环境变量**：
   - 使用 `ApiKey` 环境变量（不再需要 `DeepSeekApiKey`）
   - 删除 `ModelType` 环境变量

### 新增功能
- ✅ DeepSeek Agent 功能（工具调用）
- ✅ 支持打开终端并输出文本

### 技术改进
- 简化了代码结构
- 减少了配置复杂度
- 提高了代码可维护性

---

## v1.0.0 - 初始版本
- 支持 OpenAI 和 DeepSeek 双模型
- 基础消息处理功能
- 会话管理

