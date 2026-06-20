# MiMoTTS

MiMoTTS is a lightweight MiMo TTS gateway with a React admin dashboard, key-pool routing, upstream channel management, model mapping, call logs, and OpenAI-compatible speech API support.

> 默认后台 Token 是 `mimotts`，仅用于首次启动。生产环境请立即在「系统设置」中修改。

## Features

- Go single-binary backend + React/Vite frontend
- Docker-first deployment
- Admin dashboard for:
  - MiMo API Key pool
  - Upstream channels
  - External API Keys
  - Voice presets / voice design / voice clone
  - Call logs and statistics
  - System settings
- OpenAI-compatible endpoint: `POST /v1/audio/speech`
- Native endpoint: `POST /api/tts`
- Upstream model mapping: local standard models can be mapped to actual upstream model IDs
- Runtime admin token and outbound proxy settings
- Secret masking and explicit reveal flow

## Quick Start

```bash
git clone https://github.com/xbcvv/mimotts.git
cd mimotts
cp .env.example .env
docker compose up -d --build
```

Open:

```text
http://localhost:7117
```

Default admin token:

```text
mimotts
```

## Docker Defaults

- Container listen port: `7117`
- Host port: `7117`
- Default `ADMIN_TOKEN`: `mimotts`
- Persistent data: `./data`

## Documentation

- [Deployment](docs/deployment.md)
- [Configuration](docs/configuration.md)
- [API](docs/api.md)
- [Admin Guide](docs/admin-guide.md)
- [Upstream Model Mapping](docs/upstream-model-mapping.md)
- [Security](docs/security.md)

## Development

Backend:

```bash
cd backend
ADMIN_TOKEN=mimotts ADDR=:7117 go run .
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Full verification:

```bash
bash verify.sh
```

## License

MIT License. See [LICENSE](LICENSE).
