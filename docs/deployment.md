# Deployment

## Docker Compose

```bash
cp .env.example .env
docker compose up -d --build
```

The service listens on port `7117` inside the container and publishes `7117` on the host by default:

```yaml
ports:
  - "7117:7117"
```

Open:

```text
http://SERVER_IP:7117
```

## Persistent Data

The default compose file stores runtime data under:

```text
./data
```

This directory contains API keys, upstream channels, runtime settings, and call logs. Do not commit it.

## Production Checklist

1. Change the default admin token in **System Settings**.
2. Configure MiMo API Keys or upstream channels.
3. Create External API Keys if external callers should authenticate.
4. Set `REQUIRE_EXTERNAL_KEY=true` if unauthenticated TTS calls should be rejected.
5. Place the service behind HTTPS if exposed publicly.
