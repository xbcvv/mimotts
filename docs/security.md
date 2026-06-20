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
