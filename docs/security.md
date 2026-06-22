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
5. Regularly rotate MiMo and external keys.

## Git Hygiene

Before publishing or contributing, ensure the following are not committed:

- `.env`
- `data/`
- call logs
- real API keys
- real admin tokens
- private proxy URLs


## External API Key 明文显示规则

External API Key 当前按明文保存，管理员后台可完整显示和复制。历史版本中只保存 hash 的 Key 无法反推出明文；这类 Key 需要重新生成一次，之后即可完整显示和复制。
