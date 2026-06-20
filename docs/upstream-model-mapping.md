# Upstream Model Mapping

MiMoTTS uses local standard model IDs for internal routing:

| Local Standard Model | Meaning |
| --- | --- |
| `mimo-v2.5-tts` | Preset voice TTS |
| `mimo-v2.5-tts-voicedesign` | Voice design |
| `mimo-v2.5-tts-voiceclone` | Voice clone |

Some upstream services expose different model IDs. In the admin dashboard, click **Read Upstream Models** and map each local model to an actual upstream model.

Example:

```text
MiMo TTS（预置音色） -> provider-tts-v1
MiMo TTS（音色设计） -> provider-voice-design
MiMo TTS（音色克隆） -> provider-voice-clone
```

Docker note: if an upstream service runs on the host, `127.0.0.1` in the browser is not the same as `127.0.0.1` inside the container. MiMoTTS automatically maps localhost upstream URLs to `host.docker.internal` when running in Docker.
