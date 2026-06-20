# Configuration

## Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `ADDR` | `:7117` | Backend listen address |
| `ADMIN_TOKEN` | `mimotts` | Initial admin token |
| `DATA_DIR` | `/app/data` | Runtime data directory |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | `json` | `json` or `text` |
| `CALL_LOG_DIR` | `${DATA_DIR}/call-logs` | Call log directory |
| `REQUIRE_EXTERNAL_KEY` | `false` | Require External API Key for TTS calls |
| `CORS_ORIGIN` | empty | Optional CORS origin |

## Runtime Settings

The admin dashboard includes **System Settings**:

- Change Admin Token without editing `.env`
- Configure outbound proxy for upstream `/models` and TTS calls

Supported proxy URL examples:

```text
http://host.docker.internal:7890
socks5://host.docker.internal:7890
```

Runtime settings are saved in `${DATA_DIR}/mimotts.json`.
