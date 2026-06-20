# Admin Guide

## MiMo Keys

Manage official MiMo API keys. Keys are masked by default and require explicit reveal.

## Upstream Channels

Add OpenAI-compatible upstream services. Each channel supports:

- Base URL
- API Key
- Weight
- Enabled/disabled state
- Supported local standard models
- Model aliases to map local standard models to upstream actual model IDs

## External API Keys

Create keys for external callers. The full secret is shown only once when created.

## Voices

Supported voice types:

- Preset voice
- Voice design
- Voice clone

## System Settings

Runtime settings include:

- Admin Token
- Outbound proxy URL

Changing Admin Token takes effect immediately.


## External API Key 明文显示规则

External API Key 创建或重新生成时会显示完整值一次。系统只保存哈希，已有 External API Key 无法反查完整明文；如果忘记完整 Key，请在后台执行「重新生成 Key」，旧 Key 会立即失效。
