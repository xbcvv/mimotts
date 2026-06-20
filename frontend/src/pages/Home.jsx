import React, { useEffect, useState } from 'react'
import { Bookmark, BookmarkPlus, Download, Loader2, Plus, Save, Upload, Wand2, Volume2 } from 'lucide-react'
import { api, voices } from '../lib/api.js'

const models = [
  ['mimo-v2.5-tts', '预置音色', '选择小米内置精品音色，适合快速生成'],
  ['mimo-v2.5-tts-voicedesign', '音色设计', '用文字描述创造新的声音'],
  ['mimo-v2.5-tts-voiceclone', '音色克隆', '上传 wav/mp3 样本进行复刻']
]

function fileToDataURL(file) {
  return new Promise((ok, fail) => {
    const r = new FileReader()
    r.onload = () => ok(r.result)
    r.onerror = fail
    r.readAsDataURL(file)
  })
}

export default function Home({ auth }) {
  const [model, setModel] = useState(models[0][0])
  const [voice, setVoice] = useState('冰糖')
  const [text, setText] = useState('你好，我是 MiMoTTS。现在开始生成一段自然流畅的语音。')
  const [context, setContext] = useState('')
  const [audio, setAudio] = useState('')
  const [loading, setLoading] = useState(false)
  const [err, setErr] = useState('')
  const [voiceDataUrl, setVoiceDataUrl] = useState('')
  const [projectVoices, setProjectVoices] = useState([])
  const [saveLabel, setSaveLabel] = useState('')
  const [savedId, setSavedId] = useState('')
  const [saveHint, setSaveHint] = useState('')

  async function loadProjectVoices() {
    try {
      const data = await (await api('/api/voices')).json()
      setProjectVoices(data)
    } catch (e) { setErr(e.message) }
  }

  useEffect(() => { loadProjectVoices() }, [])

  async function submit() {
    setErr('')
    setLoading(true)
    if (audio) URL.revokeObjectURL(audio)
    setAudio('')
    try {
      const res = await api('/api/tts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model, voice, text, context, voiceDataUrl })
      })
      const blob = await res.blob()
      setAudio(URL.createObjectURL(blob))
    } catch (e) { setErr(e.message) }
    finally { setLoading(false) }
  }

  async function saveAsVoice() {
    setSaveHint('')
    setSavedId('')
    if (model === 'mimo-v2.5-tts') {
      setSaveHint('预置音色请使用管理页"音色库"创建')
      return
    }
    if (model === 'mimo-v2.5-tts-voicedesign' && !context.trim()) {
      setSaveHint('请先填写音色描述')
      return
    }
    if (model === 'mimo-v2.5-tts-voiceclone' && !voiceDataUrl) {
      setSaveHint('请先上传音色样本')
      return
    }
    try {
      const body = {
        kind: model === 'mimo-v2.5-tts-voicedesign' ? 'design' : 'clone',
        label: saveLabel || (model === 'mimo-v2.5-tts-voicedesign' ? '我的设计音色' : '我的克隆音色'),
        design: model === 'mimo-v2.5-tts-voicedesign' ? context : undefined,
        cloneDataUrl: model === 'mimo-v2.5-tts-voiceclone' ? voiceDataUrl : undefined,
        language: 'zh'
      }
      const v = await (await api('/api/admin/voices', {
        method: 'POST',
        headers: auth,
        body: JSON.stringify(body)
      })).json()
      setSavedId(v.id)
      setSaveLabel('')
      setSaveHint('✓ 已保存为项目音色')
      await loadProjectVoices()
    } catch (e) { setSaveHint('保存失败：' + e.message) }
  }

  function pickProjectVoice(id) {
    const v = projectVoices.find(x => x.id === id)
    if (!v) return
    setVoice(v.id)
    if (v.kind === 'design') {
      setModel('mimo-v2.5-tts-voicedesign')
      setContext(v.design || '')
    } else if (v.kind === 'clone') {
      setModel('mimo-v2.5-tts-voiceclone')
      setVoiceDataUrl('__stored__')
    } else if (v.kind === 'preset') {
      setModel('mimo-v2.5-tts')
    }
  }

  return (
    <div>
      {/* Section header */}
      <div className="section-header mb-4">
        <h1>语音合成</h1>
        <p>预置音色、音色设计、音色克隆，一站式生成 WAV 语音。MiMo Key 多 Key 自动轮询。</p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr', gap: 20 }}>
        {/* Text Input Card */}
        <div className="card-panel">
          <div className="card-panel-title">合成文本</div>
          <div className="card-panel-desc">输入需要合成的文字内容，支持中英文混合，最多 2500 字。</div>
          <textarea
            className="big"
            value={text}
            onChange={e => setText(e.target.value)}
            maxLength={2500}
            placeholder="在此输入要合成的文本..."
          />
          <div className="flex" style={{ justifyContent: 'flex-end' }}>
            <span className="text-muted text-sm">{text.length} / 2500</span>
          </div>
        </div>

        {/* Model + Voice Selection */}
        <div className="card-panel">
          <div className="card-panel-title">模型与音色</div>
          <div className="card-panel-desc">选择合成模型模式和对应音色</div>

          <label className="field-label">模型模式</label>
          <div className="modelgrid">
            {models.map(m => (
              <button
                key={m[0]}
                className={model === m[0] ? 'card selected' : 'card'}
                onClick={() => setModel(m[0])}
              >
                <b>{m[1]}</b>
                <span>{m[2]}</span>
              </button>
            ))}
          </div>

          {model === 'mimo-v2.5-tts' && (
            <>
              <label className="field-label">预置音色</label>
              <div className="voicegrid">
                {voices.map(v => (
                  <button
                    key={v[0]}
                    className={voice === v[0] ? 'voice selected' : 'voice'}
                    onClick={() => setVoice(v[0])}
                  >
                    <b>{v[0]}</b>
                    <span>{v[1]}</span>
                  </button>
                ))}
              </div>
            </>
          )}

          {model === 'mimo-v2.5-tts-voicedesign' && (
            <>
              <label className="field-label">音色描述 / 导演模式</label>
              <textarea
                value={context}
                onChange={e => setContext(e.target.value)}
                placeholder="例如：青年男性，电竞解说风格，语速极快且连贯..."
              />
            </>
          )}

          {model !== 'mimo-v2.5-tts-voicedesign' && (
            <>
              <label className="field-label">风格控制（可选）</label>
              <input
                value={context}
                onChange={e => setContext(e.target.value)}
                placeholder="例如：温柔、语速稍慢、带一点笑意"
              />
            </>
          )}

          {model === 'mimo-v2.5-tts-voiceclone' && (
            <label className="upload mt-4">
              <Upload size={16} /> 上传 wav/mp3 音色样本
              <input
                type="file"
                accept="audio/wav,audio/mpeg"
                onChange={async e => setVoiceDataUrl(await fileToDataURL(e.target.files[0]))}
              />
              {voiceDataUrl && <span>✓ 已选择样本</span>}
            </label>
          )}
        </div>

        {/* Project Voices */}
        {projectVoices.length > 0 && (
          <div className="card-panel">
            <div className="card-panel-title">项目音色</div>
            <div className="card-panel-desc">已保存的项目级音色，外部 API 可通过 voice 字段调用</div>
            <div className="voicegrid">
              {projectVoices.map(v => (
                <button
                  key={v.id}
                  className={voice === v.id ? 'voice selected' : 'voice'}
                  onClick={() => pickProjectVoice(v.id)}
                >
                  <b>{v.label}</b>
                  <code className="vid">{v.id}</code>
                  <span>{v.kind} · {v.language}</span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Generate */}
        <div className="card-panel" style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <button className="primary" onClick={submit} disabled={loading}>
            {loading ? <Loader2 className="spin" size={18} /> : <Wand2 size={18} />}
            {loading ? '合成中...' : '生成语音'}
          </button>
          {(model === 'mimo-v2.5-tts-voicedesign' || model === 'mimo-v2.5-tts-voiceclone') && (
            <>
              <input
                value={saveLabel}
                onChange={e => setSaveLabel(e.target.value)}
                placeholder="给这个音色起个名字"
                style={{ maxWidth: 240 }}
              />
              <button className="btn" onClick={saveAsVoice}>
                <Save size={16} /> 保存为项目音色
              </button>
            </>
          )}
          {saveHint && (
            <span className={savedId ? 'toast' : 'hint'}>
              {saveHint}{savedId && <> · voiceId = <code>{savedId}</code></>}
            </span>
          )}
          {err && <span className="error">{err}</span>}
        </div>

        {/* Result */}
        {audio && (
          <div className="player">
            <Volume2 size={20} style={{ color: 'var(--primary)' }} />
            <audio src={audio} controls autoPlay />
            <a href={audio} download="mimotts.wav">
              <Download size={16} /> 下载 WAV
            </a>
          </div>
        )}
      </div>
    </div>
  )
}
