# MiMoTTS

MiMoTTS 是一个轻量级 MiMo TTS 网关，提供 React 管理后台、MiMo Key 池、上游渠道路由、模型映射、调用日志，以及兼容 OpenAI `/v1/audio/speech` 的语音合成接口。

> 默认后台 Token 是 `mimotts`，仅用于首次启动。生产环境请登录后立即在「系统设置」中修改。

## 功能特性

- Go 单体后端 + React/Vite 前端
- Docker 一键部署
- 管理后台支持：
  - MiMo API Key 池管理
  - 上游渠道管理
  - External API Key 管理
  - 预置音色 / 音色设计 / 音色克隆
  - 调用日志与统计
  - 系统设置
- 兼容 OpenAI 的语音接口：`POST /v1/audio/speech`
- 原生语音接口：`POST /api/tts`
- 支持上游模型映射：可将本地标准模型映射到上游实际模型 ID
- 支持运行时修改 Admin Token
- 支持配置出站代理
- 敏感字段默认脱敏，完整密钥需要显式显示

## 快速开始

```bash
git clone https://github.com/xbcvv/mimotts.git
cd mimotts
cp .env.example .env
docker compose up -d --build
```

访问：

```text
http://localhost:7117
```

默认后台 Token：

```text
mimotts
```

## Docker 默认配置

- 容器内监听端口：`7117`
- 宿主机默认端口：`7117`
- 默认 `ADMIN_TOKEN`：`mimotts`
- 默认数据目录：`./data`

`docker-compose.yml` 默认端口映射：

```yaml
ports:
  - "7117:7117"
```

## 项目结构

```text
mimotts/
├── backend/                 # Go 后端
│   ├── calllog/             # 调用日志
│   ├── slogx/               # 日志脱敏与格式化
│   ├── store/               # JSON 数据存储
│   ├── config.go            # 配置加载
│   ├── mimo.go              # MiMo TTS 调用逻辑
│   ├── routes.go            # API 路由
│   └── spa.go               # 前端静态文件托管
├── frontend/                # React/Vite 前端
│   └── src/
├── docs/                    # 详细文档
├── Dockerfile
├── docker-compose.yml
├── .env.example
└── verify.sh
```

## 管理后台

管理后台包含以下页面：

- 语音合成
- 音色管理
- MiMo Keys
- 上游渠道
- API Keys
- 调用记录
- 系统状态
- 系统设置

### 系统设置

系统设置支持：

- 修改后台 Admin Token
- 配置出站代理

代理用于：

- 读取上游 `/models`
- 实际 TTS 上游调用

代理示例：

```text
http://host.docker.internal:7890
socks5://host.docker.internal:7890
```

如果代理运行在宿主机，Docker 容器内建议使用 `host.docker.internal`，不要使用容器内的 `127.0.0.1`。

## 上游渠道与模型映射

MiMoTTS 内部使用本地标准模型：

| 本地标准模型 | 说明 |
| --- | --- |
| `mimo-v2.5-tts` | 预置音色 TTS |
| `mimo-v2.5-tts-voicedesign` | 音色设计 |
| `mimo-v2.5-tts-voiceclone` | 音色克隆 |

如果你的上游服务暴露的是其它模型 ID，可以在「上游渠道」中点击「读取上游模型」，然后建立映射关系。

示例：

```text
MiMo TTS（预置音色） -> provider-tts-v1
MiMo TTS（音色设计） -> provider-voice-design
MiMo TTS（音色克隆） -> provider-voice-clone
```

这样外部调用仍然可以使用 MiMoTTS 的标准模型，实际请求会自动转发到上游对应模型。

## API 使用

### 原生 TTS 接口

```http
POST /api/tts
Content-Type: application/json
```

请求示例：

```json
{
  "text": "你好，我是 MiMoTTS。",
  "model": "mimo-v2.5-tts",
  "voice": "白桦"
}
```

返回：`audio/wav`。

### OpenAI 兼容语音接口

```http
POST /v1/audio/speech
Content-Type: application/json
```

请求示例：

```json
{
  "model": "mimo-v2.5-tts",
  "input": "你好，我是 MiMoTTS。",
  "voice": "白桦",
  "response_format": "wav"
}
```

返回：`audio/wav`。

### 外部 API Key

如果启用了：

```env
REQUIRE_EXTERNAL_KEY=true
```

调用 TTS 接口时需要携带 External API Key：

```http
X-API-Key: <external-api-key>
```

或：

```http
Authorization: Bearer <external-api-key>
```

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:7117` | 后端监听地址 |
| `ADMIN_TOKEN` | `mimotts` | 初始后台 Token |
| `DATA_DIR` | `/app/data` | 运行数据目录 |
| `LOG_LEVEL` | `info` | 日志级别：`debug` / `info` / `warn` / `error` |
| `LOG_FORMAT` | `json` | 日志格式：`json` / `text` |
| `CALL_LOG_DIR` | `${DATA_DIR}/call-logs` | 调用日志目录 |
| `REQUIRE_EXTERNAL_KEY` | `false` | 是否要求 TTS 调用必须携带 External API Key |
| `CORS_ORIGIN` | 空 | 可选跨域来源 |

## 本地开发

### 后端

```bash
cd backend
ADMIN_TOKEN=mimotts ADDR=:7117 go run .
```

### 前端

```bash
cd frontend
npm install
npm run dev
```

### 完整验证

```bash
bash verify.sh
```

验证内容包括：

- Go 单元测试
- Go 构建
- `go vet`
- 前端构建

## 安全建议

公开部署前建议：

1. 登录后立即修改默认 Admin Token。
2. 不要提交 `.env`、`data/`、调用日志或真实密钥。
3. 如果暴露到公网，建议放到 HTTPS 反向代理后面。
4. 如果提供给外部系统调用，建议开启 `REQUIRE_EXTERNAL_KEY=true`。
5. 定期轮换 MiMo Key、上游渠道 Key 和 External API Key。

## 详细文档

- [部署说明](docs/deployment.md)
- [配置说明](docs/configuration.md)
- [API 文档](docs/api.md)
- [后台使用指南](docs/admin-guide.md)
- [上游模型映射](docs/upstream-model-mapping.md)
- [安全说明](docs/security.md)

## License

本项目使用 MIT License，详见 [LICENSE](LICENSE)。


## External API Key 明文显示规则

External API Key 当前按明文保存，管理员后台可完整显示和复制。历史版本中只保存 hash 的 Key 无法反推出明文；这类 Key 需要重新生成一次，之后即可完整显示和复制。
