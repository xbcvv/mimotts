# Security

## Default Token

The default `ADMIN_TOKEN` is:

```text
mimotts
```

Change it immediately after first login.

## Secrets

- MiMo API Keys are masked by default.
- External API Key full secret is shown only once at creation.
- Runtime data is stored under `DATA_DIR` and should never be committed.

## Public Exposure

If exposing MiMoTTS to the Internet:

1. Use HTTPS.
2. Change Admin Token.
3. Enable `REQUIRE_EXTERNAL_KEY=true` for external TTS calls.
4. Restrict access with a reverse proxy or firewall when possible.
5. Regularly rotate upstream and external keys.

## Git Hygiene

Before publishing or contributing, ensure the following are not committed:

- `.env`
- `data/`
- call logs
- real API keys
- real admin tokens
- private proxy URLs


## External API Key 明文显示规则

External API Key 创建或重新生成时会显示完整值一次。系统只保存哈希，已有 External API Key 无法反查完整明文；如果忘记完整 Key，请在后台执行「重新生成 Key」，旧 Key 会立即失效。
