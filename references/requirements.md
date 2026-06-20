# MimoTTS 项目需求

## 参考
- 前端复刻目标：https://tts.cngov.cc.cd/
- 小米 TTS 文档：https://platform.xiaomimimo.com/docs/zh-CN/usage-guide/speech-synthesis-v2.5
- 首次 API 调用：https://platform.xiaomimimo.com/docs/zh-CN/quick-start/first-api-call
- 核心参考：https://github.com/XiaomiMiMo/MiMo-Skills

## 技术栈
- 后端：Express
- 前端：React + Vite
- UI 图标：lucide-react

## 功能目标
1. 补全前端 TTS 调用页面，尽量复刻目标站点交互。
2. 补全 Express 后端：MiMo TTS 代理调用、音频返回、配置读取。
3. 增加前端管理员页面：
   - 写入/删除/查看 MiMo API Key（禁止明文回显，只显示掩码）
   - 外部调用 Key 管理：创建、删除、启用/禁用、列表
   - 后端配置/健康状态查看
4. API 安全：
   - 前端管理员入口需要 admin token 或密码保护
   - 外部调用接口需要服务端生成的外部 API Key
   - key 持久化到本地数据文件或 sqlite，不提交真实 key
5. 输出要求：
   - 能本地 npm install / npm run dev 启动
   - README 写清启动、配置、接口、部署注意事项
   - 不破坏其他项目

## MiMo 调用关键点
- OpenAI 兼容 base_url: https://api.xiaomimimo.com/v1
- TTS 文本需放在 role=assistant 的 messages 中
- user role 可放自然语言风格控制
- 模型：mimo-v2.5-tts / mimo-v2.5-tts-voicedesign / mimo-v2.5-tts-voiceclone
