# Admin Guide

## MiMo Keys

Manage official MiMo API keys. Keys are masked by default and require explicit reveal.

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

External API Key 当前按明文保存，管理员后台可完整显示和复制。历史版本中只保存 hash 的 Key 无法反推出明文；这类 Key 需要重新生成一次，之后即可完整显示和复制。
