# API

## Authentication

Admin APIs require:

```http
Authorization: Bearer <ADMIN_TOKEN>
```

TTS APIs can optionally require External API Keys when `REQUIRE_EXTERNAL_KEY=true`:

```http
X-API-Key: <external-api-key>
```

or:

```http
Authorization: Bearer <external-api-key>
```

## Native TTS

```http
POST /api/tts
Content-Type: application/json
```

```json
{
  "text": "你好，我是 MiMoTTS。",
  "model": "mimo-v2.5-tts",
  "voice": "白桦"
}
```

Returns `audio/wav`.

## OpenAI-Compatible Speech

```http
POST /v1/audio/speech
Content-Type: application/json
```

```json
{
  "model": "mimo-v2.5-tts",
  "input": "你好，我是 MiMoTTS。",
  "voice": "白桦",
  "response_format": "wav"
}
```

Returns `audio/wav`.

## Public Voices

```http
GET /api/voices
```

Returns project voices and presets that can be used by callers.
