import React, { useEffect, useState } from 'react'
import {
  Activity, Copy, Eye, EyeOff, KeyRound, Mic, MoreHorizontal, Pencil, Plus, RefreshCw, Search, Settings, ShieldCheck, Trash2
} from 'lucide-react'
import { api } from '../lib/api.js'

export default function Admin({ page, auth }) {
  const [msg, setMsg] = useState('')
  const [err, setErr] = useState('')

  // Auto-dismiss messages
  useEffect(() => {
    if (msg) { const t = setTimeout(() => setMsg(''), 3000); return () => clearTimeout(t) }
  }, [msg])

  return (
    <div>
      {msg && <div className="toast mb-4" onClick={() => setMsg('')}>{msg}</div>}
      {err && <div className="error mb-4" onClick={() => setErr('')}>{err}</div>}

      {page === 'status'   && <Status auth={auth} setErr={setErr} />}
      {page === 'mimo'     && <MiMo auth={auth} setMsg={setMsg} setErr={setErr} />}
      {page === 'voices'   && <Voices auth={auth} setMsg={setMsg} setErr={setErr} />}
      {page === 'ext'      && <External auth={auth} setMsg={setMsg} setErr={setErr} />}
      {page === 'calllogs' && <CallLogs auth={auth} setMsg={setMsg} setErr={setErr} />}
      {page === 'settings' && <SettingsPage auth={auth} setMsg={setMsg} setErr={setErr} />}
    </div>
  )
}

/* ====================== Status ====================== */
function Status({ auth, setErr }) {
  const [data, setData] = useState(null)

  async function load() {
    try {
      setData(await (await api('/api/admin/status', { headers: auth })).json())
    } catch (e) { setErr(e.message) }
  }

  useEffect(() => { load() }, [])

  const items = statusItemsZh(data)

  return (
    <div>
      <div className="section-header mb-4">
        <h1><ShieldCheck size={20} style={{ display: 'inline', verticalAlign: 'middle' }} /> 系统状态</h1>
        <p>查看服务运行状态和统计数据</p>
      </div>

      <div className="card-panel">
        <div className="flex" style={{ justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
          <span className="card-panel-title">运行状态</span>
          <button className="smallbtn" onClick={load}>
            <RefreshCw size={14} /> 刷新
          </button>
        </div>

        {data ? (
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: 10 }}>
            {items.map(([k, v]) => (
              <div key={k} style={{
                padding: '12px 16px',
                background: 'var(--bg-sub)',
                borderRadius: 'var(--radius)',
                border: '1px solid var(--card-border)'
              }}>
                <div className="text-muted text-sm" style={{ textTransform: 'uppercase', letterSpacing: '.03em', fontSize: '.72rem' }}>{k}</div>
                <div className="mono" style={{ fontWeight: 600, marginTop: 2 }}>
                  {typeof v === 'boolean' ? (v ? '✓' : '✗') : String(v)}
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-muted">加载中...</p>
        )}
      </div>
    </div>
  )
}

/* ====================== Secret Field ====================== */
function SecretField({ value }) {
  if (!value) return null
  return <pre className="reveal-secret">{value}</pre>
}


/* ====================== Shared Admin List UI ====================== */
function maskSecret(v) { if (!v) return '未配置'; const t = String(v); return t.includes('*') ? t : (t.length <= 10 ? '**********' : `${t.slice(0,4)}••••••••${t.slice(-4)}`) }
function copyText(text, setMsg) { if (!text) return; navigator.clipboard?.writeText(text).then(() => setMsg?.('已复制')).catch(() => {}) }
function PageShell({ icon, title, desc, count, activeCount, action, children }) {
  return <div className="list-page"><div className="list-page-head"><div><h1>{icon} {title}</h1><p>{desc}</p></div><div className="list-page-actions"><span className="metric-pill">全部 {count}</span><span className="metric-pill ok">启用 {activeCount}</span>{action}</div></div>{children}</div>
}
function SegmentTabs({ value, onChange, tabs }) { return <div className="segment-tabs">{tabs.map(t => <button key={t.value} className={value === t.value ? 'active' : ''} onClick={() => onChange(t.value)}>{t.label}</button>)}</div> }
function ListToolbar({ search, setSearch, status, setStatus, placeholder }) {
  return <div className="list-toolbar"><div className="search-box"><Search size={14}/><input value={search} onChange={e=>setSearch(e.target.value)} placeholder={placeholder}/></div><select value={status} onChange={e=>setStatus(e.target.value)}><option value="all">全部状态</option><option value="enabled">已启用</option><option value="disabled">已禁用</option></select></div>
}
function RowMenu({ open, children }) { return open ? <div className="row-menu">{children}</div> : null }
function statusItemsZh(data) {
  if (!data) return []
  const stats = data.stats || {}
  return [
    ['MiMo Key 总数', data.mimoKeyCount],
    ['MiMo 可用 Key', data.mimoUsableCount],
    ['外部 API Key 数量', data.externalKeyCount],
    ['项目音色数量', data.voiceCount],
    ['累计调用次数', stats.totalCalls],
    ['累计失败次数', stats.failedCalls],
    ['最近调用时间', stats.lastCallAt ? new Date(stats.lastCallAt).toLocaleString() : '—'],
    ['最近失败时间', stats.lastFailedAt ? new Date(stats.lastFailedAt).toLocaleString() : '—'],
    ['累计轮询次数', stats.keyRoundsTotal],
  ]
}
function standardModelLabel(id) {
  return ({'mimo-v2.5-tts':'MiMo TTS（预置音色）','mimo-v2.5-tts-voiceclone':'MiMo TTS（音色克隆）','mimo-v2.5-tts-voicedesign':'MiMo TTS（音色设计）','*':'全部模型'})[id] || id
}

/* ====================== MiMo Keys ====================== */
function MiMo({ auth, setMsg, setErr }) {
  const [items, setItems] = useState([]), [label, setLabel] = useState('主 Key'), [key, setKey] = useState(''), [weight, setWeight] = useState(1)
  const [editing, setEditing] = useState(null), [editLabel, setEditLabel] = useState(''), [editKey, setEditKey] = useState(''), [editWeight, setEditWeight] = useState(1)
  const [revealed, setRevealed] = useState({}), [showCreate, setShowCreate] = useState(false), [search, setSearch] = useState(''), [status, setStatus] = useState('all'), [tab, setTab] = useState('all'), [menu, setMenu] = useState(null)
  async function load(){ try{ setItems(await (await api('/api/admin/mimo-keys',{headers:auth})).json()) }catch(e){setErr(e.message)} }
  useEffect(()=>{load()},[])
  async function add(){ try{ await api('/api/admin/mimo-keys',{method:'POST',headers:auth,body:JSON.stringify({label,key,weight})}); setKey(''); setShowCreate(false); setMsg('已添加 MiMo Key'); await load() }catch(e){setErr(e.message)} }
  async function del(id){ if(!confirm('确认删除这个 MiMo Key？'))return; try{ await api(`/api/admin/mimo-keys/${id}`,{method:'DELETE',headers:auth}); await load() }catch(e){setErr(e.message)} }
  async function toggle(id,en){ try{ await api(`/api/admin/mimo-keys/${id}`,{method:'PATCH',headers:auth,body:JSON.stringify({enabled:!en})}); await load() }catch(e){setErr(e.message)} }
  function startEdit(x){ setEditing(editing===x.id?null:x.id); setEditLabel(x.label); setEditKey(''); setEditWeight(x.weight); setMenu(null) }
  async function saveEdit(id){ try{ const body={label:editLabel,weight:editWeight}; if(editKey.trim()) body.key=editKey; await api(`/api/admin/mimo-keys/${id}`,{method:'PATCH',headers:auth,body:JSON.stringify(body)}); setEditing(null); setMsg('MiMo Key 已更新'); await load() }catch(e){setErr(e.message)} }
  async function reveal(id){ if(revealed[id]){setRevealed(r=>({...r,[id]:''}));return} if(!confirm('完整 Key 属于敏感信息，确认显示？'))return; try{const d=await (await api(`/api/admin/mimo-keys/${id}/secret`,{headers:auth})).json(); setRevealed(r=>({...r,[id]:d.secret||''}))}catch(e){setErr(e.message)} }
  const rows=items.filter(x=>{const q=search.trim().toLowerCase(); return (!q||[x.id,x.label,x.masked].filter(Boolean).some(v=>String(v).toLowerCase().includes(q)))&&(status==='all'||(status==='enabled'?x.enabled:!x.enabled))&&(tab==='all'||(tab==='active'?x.enabled:!x.enabled))})
  return <PageShell icon={<KeyRound size={22}/>} title="MiMo API Keys" desc="管理官方 MiMo API Key 池，按权重轮询调用。" count={items.length} activeCount={items.filter(x=>x.enabled).length} action={<button className="orange-btn" onClick={()=>setShowCreate(!showCreate)}><Plus size={15}/> 创建 MiMo Key</button>}>
    <SegmentTabs value={tab} onChange={setTab} tabs={[{value:'all',label:'全部'},{value:'active',label:'已启用'},{value:'disabled',label:'已禁用'}]}/>
    {showCreate&&<div className="drawer-card"><div className="drawer-title">新增 MiMo Key</div><div className="drawer-grid"><label>名称<input value={label} onChange={e=>setLabel(e.target.value)}/></label><label className="wide">API Key<input value={key} onChange={e=>setKey(e.target.value)} type="password"/></label><label>权重<input type="number" min="1" max="100" value={weight} onChange={e=>setWeight(Number(e.target.value))}/></label><button className="primary" onClick={add}>保存</button></div></div>}
    <ListToolbar search={search} setSearch={setSearch} status={status} setStatus={setStatus} placeholder="按名称、ID、Key 搜索..."/>
    <div className="table-card table-card-open"><table className="admin-table"><thead><tr><th>ID</th><th>名称</th><th>API KEY</th><th>权重</th><th>类型</th><th>状态</th><th>操作</th></tr></thead><tbody>{rows.map(x=><React.Fragment key={x.id}><tr><td className="mono muted">#{x.id}</td><td><b>{x.label}</b></td><td><div className="secret-cell"><span>{revealed[x.id]||maskSecret(x.masked)}</span><button onClick={()=>reveal(x.id)}>{revealed[x.id]?<EyeOff size={14}/>:<Eye size={14}/>}</button><button onClick={()=>copyText(revealed[x.id]||x.masked,setMsg)}><Copy size={14}/></button></div></td><td>{x.weight}</td><td><span className="blue-link">MiMo</span></td><td><span className={x.enabled?'status-text ok':'status-text off'}>{x.enabled?'已启用':'已禁用'}</span></td><td className="op-cell"><button className="dots" onClick={()=>setMenu(menu===x.id?null:x.id)}><MoreHorizontal size={18}/></button><RowMenu open={menu===x.id}><button onClick={()=>startEdit(x)}><Pencil size={14}/> 编辑</button><button onClick={()=>toggle(x.id,x.enabled)}><Settings size={14}/> {x.enabled?'禁用':'启用'}</button><button onClick={()=>reveal(x.id)}><Eye size={14}/> {revealed[x.id]?'隐藏密钥':'显示密钥'}</button><button className="danger" onClick={()=>del(x.id)}><Trash2 size={14}/> 删除</button></RowMenu></td></tr>{editing===x.id&&<tr className="edit-row"><td colSpan="7"><div className="inline-edit"><label>名称<input value={editLabel} onChange={e=>setEditLabel(e.target.value)}/></label><label className="wide">API Key<input value={editKey} onChange={e=>setEditKey(e.target.value)} placeholder="留空不修改" type="password"/></label><label>权重<input type="number" min="1" max="100" value={editWeight} onChange={e=>setEditWeight(Number(e.target.value))}/></label><button className="primary" onClick={()=>saveEdit(x.id)}>保存</button><button className="btn" onClick={()=>setEditing(null)}>取消</button></div></td></tr>}</React.Fragment>)}{rows.length===0&&<tr><td colSpan="7" className="empty-cell">暂无 MiMo Key</td></tr>}</tbody></table></div>
  </PageShell>
}

/* ====================== Voices ====================== */
function Voices({ auth, setMsg, setErr }) {
  const [items, setItems] = useState([])
  const [kind, setKind] = useState('preset')
  const [label, setLabel] = useState('')
  const [preset, setPreset] = useState('冰糖')
  const [design, setDesign] = useState('')
  const [cloneDataURL, setCloneDataURL] = useState('')
  const [language, setLanguage] = useState('zh')
  const [desc, setDesc] = useState('')

  async function load() {
    try { setItems(await (await api('/api/admin/voices', { headers: auth })).json()) }
    catch (e) { setErr(e.message) }
  }
  useEffect(() => { load() }, [])

  async function add() {
    try {
      await api('/api/admin/voices', { method: 'POST', headers: auth, body: JSON.stringify({ kind, label, preset, design, cloneDataUrl: cloneDataURL, language, description: desc }) })
      setLabel(''); setDesign(''); setCloneDataURL(''); setDesc(''); setMsg('音色已创建'); await load()
    } catch (e) { setErr(e.message) }
  }

  async function del(id) {
    try { await api(`/api/admin/voices/${id}`, { method: 'DELETE', headers: auth }); await load() }
    catch (e) { setErr(e.message) }
  }

  function readFile(e) {
    const f = e.target.files[0]
    if (!f) return
    const r = new FileReader()
    r.onload = () => setCloneDataURL(r.result)
    r.readAsDataURL(f)
  }

  return (
    <div>
      <div className="section-header mb-4">
        <h1><Mic size={20} style={{ display: 'inline', verticalAlign: 'middle' }} /> 音色管理</h1>
        <p>项目级音色 ID，外部 API 通过 voice 字段直接调用</p>
      </div>

      <div className="card-panel">
        <div className="card-title"><b>新增音色</b></div>
        <div className="row">
          <select value={kind} onChange={e => setKind(e.target.value)}>
            <option value="preset">预置音色</option>
            <option value="design">音色设计</option>
            <option value="clone">音色克隆</option>
          </select>
          <input value={label} onChange={e => setLabel(e.target.value)} placeholder="音色名称" />
          <select value={language} onChange={e => setLanguage(e.target.value)}>
            <option value="zh">中文</option>
            <option value="en">English</option>
          </select>
        </div>
        {kind === 'preset' && (
          <div className="row">
            <select value={preset} onChange={e => setPreset(e.target.value)}>
              {Object.entries({ '冰糖': '活泼少女', '茉莉': '知性女声', '苏打': '阳光少年', '白桦': '成熟男声', 'Mia': 'Lively girl', 'Chloe': 'Sweet Dreamy', 'Milo': 'Sunny boy', 'Dean': 'Steady Gentle' }).map(([k, v]) => <option key={k} value={k}>{k} ({v})</option>)}
            </select>
          </div>
        )}
        {kind === 'design' && (
          <textarea value={design} onChange={e => setDesign(e.target.value)} placeholder="音色描述：例如 中年男性，拍卖师风格，节奏极快..." />
        )}
        {kind === 'clone' && (
          <label className="upload"><Mic size={16} /> 上传 wav/mp3
            <input type="file" accept="audio/wav,audio/mpeg" onChange={readFile} />
            {cloneDataURL && <span>✓ 已选择</span>}
          </label>
        )}
        <input value={desc} onChange={e => setDesc(e.target.value)} placeholder="备注（可选）" className="mt-2" />
        <button className="primary mt-4" onClick={add}><Plus size={16} /> 创建音色</button>
      </div>

      <div className="config-grid">
        {items.map(v => (
          <div className="config-card" key={v.id}>
            <div className="card-head">
              <div>
                <h3>{v.label}</h3>
                <code>{v.id}</code>
              </div>
              <span className="badge ok">{v.kind}</span>
            </div>
            <div className="card-body">
              <div className="kv"><span>类型</span><b>{v.kind}</b></div>
              <div className="kv"><span>语言</span><b>{v.language}</b></div>
              {v.description && <div className="kv"><span>备注</span><b>{v.description}</b></div>}
            </div>
            <div className="card-actions">
              <button className="danger-btn" onClick={() => del(v.id)}><Trash2 size={16} /> 删除</button>
            </div>
          </div>
        ))}
        {items.length === 0 && <p className="text-muted text-sm" style={{ padding: 16 }}>暂无项目音色</p>}
      </div>
    </div>
  )
}


/* ====================== External Keys ====================== */
function External({ auth, setMsg, setErr }) {
  const [items,setItems]=useState([]),[label,setLabel]=useState('应用A'),[secret,setSecret]=useState(''),[showCreate,setShowCreate]=useState(false),[search,setSearch]=useState(''),[status,setStatus]=useState('all'),[tab,setTab]=useState('all'),[menu,setMenu]=useState(null),[editing,setEditing]=useState(null),[editLabel,setEditLabel]=useState(''),[editPermissions,setEditPermissions]=useState(['tts']),[rotatedSecret,setRotatedSecret]=useState(null)
  async function load(){try{setItems(await (await api('/api/admin/ext-keys',{headers:auth})).json())}catch(e){setErr(e.message)}}
  useEffect(()=>{load()},[])
  async function add(){try{const d=await (await api('/api/admin/ext-keys',{method:'POST',headers:auth,body:JSON.stringify({label,permissions:['tts']})})).json();setSecret(d.secret);setShowCreate(false);setMsg('外部 Key 已创建');await load()}catch(e){setErr(e.message)}}
  async function del(id){if(!confirm('确认删除这个外部 Key？'))return;try{await api(`/api/admin/ext-keys/${id}`,{method:'DELETE',headers:auth});await load()}catch(e){setErr(e.message)}}
  async function toggle(x){try{await api(`/api/admin/ext-keys/${x.id}`,{method:'PATCH',headers:auth,body:JSON.stringify({enabled:!x.enabled})});await load()}catch(e){setErr(e.message)}}
  function startEdit(x){ setEditing(editing===x.id?null:x.id); setEditLabel(x.label||''); setEditPermissions(x.permissions||['tts']); setMenu(null) }
  async function saveEdit(id){ try{ await api(`/api/admin/ext-keys/${id}`,{method:'PATCH',headers:auth,body:JSON.stringify({label:editLabel,permissions:editPermissions})}); setEditing(null); setMsg('API Key 已更新'); await load() }catch(e){setErr(e.message)} }
  async function rotateKey(x){ if(!confirm('重新生成后，旧 Key 会立即失效。新完整 Key 只显示一次，确认继续？')) return; try{ const d=await (await api(`/api/admin/ext-keys/${x.id}/rotate`,{method:'POST',headers:auth})).json(); setRotatedSecret({id:x.id, label:x.label, secret:d.secret}); setMsg('已重新生成 API Key，请立即复制保存'); await load() }catch(e){setErr(e.message)} }
  const rows=items.filter(x=>{const q=search.trim().toLowerCase();return(!q||[x.id,x.label,x.masked,x.prefix].filter(Boolean).some(v=>String(v).toLowerCase().includes(q)))&&(status==='all'||(status==='enabled'?x.enabled:!x.enabled))&&(tab==='all'||(tab==='active'?x.enabled:!x.enabled))})
  return <PageShell icon={<KeyRound size={22}/>} title="API 密钥" desc="创建和管理用于 API 认证的访问密钥。" count={items.length} activeCount={items.filter(x=>x.enabled).length} action={<button className="orange-btn" onClick={()=>setShowCreate(!showCreate)}><Plus size={15}/> 创建 API Key</button>}>
    <SegmentTabs value={tab} onChange={setTab} tabs={[{value:'all',label:'全部'},{value:'active',label:'用户'},{value:'disabled',label:'已禁用'}]}/>
    {showCreate&&<div className="drawer-card"><div className="drawer-title">创建 API Key</div><div className="drawer-grid"><label className="wide">名称<input value={label} onChange={e=>setLabel(e.target.value)} placeholder="例如：tts / codex / 交易员"/></label><button className="primary" onClick={add}>创建</button></div></div>}
    {secret&&<div className="secret-once"><b>请立即复制，此 Key 只显示一次：</b><code>{secret}</code><button className="smallbtn" onClick={()=>copyText(secret,setMsg)}><Copy size={14}/> 复制</button></div>}
    <ListToolbar search={search} setSearch={setSearch} status={status} setStatus={setStatus} placeholder="按名称或 API Key 搜索..."/>
    <div className="table-card table-card-open"><table className="admin-table"><thead><tr><th></th><th>ID</th><th>名称</th><th>API KEY</th><th>创建者</th><th>类型</th><th>状态</th><th>创建时间</th><th>更新时间</th><th>操作</th></tr></thead><tbody>{rows.map(x=><React.Fragment key={x.id}><tr><td><input type="checkbox"/></td><td className="mono muted">#{x.id}</td><td><b>{x.label}</b></td><td><div className="secret-cell"><span>{maskSecret(x.masked||x.prefix)}</span><button onClick={()=>copyText(x.masked||x.prefix,setMsg)}><Copy size={14}/></button></div></td><td>{x.createdBy||'system'}</td><td><span className="blue-link">用户</span></td><td><span className={x.enabled?'status-text ok':'status-text off'}>{x.enabled?'已启用':'已禁用'}</span></td><td className="mono muted">{x.createdAt?new Date(x.createdAt).toLocaleString():'—'}</td><td className="mono muted">{x.updatedAt?new Date(x.updatedAt).toLocaleString():'—'}</td><td className="op-cell"><button className="dots" onClick={()=>setMenu(menu===x.id?null:x.id)}><MoreHorizontal size={18}/></button><RowMenu open={menu===x.id}><button onClick={()=>startEdit(x)}><Pencil size={14}/> 编辑</button><button onClick={()=>toggle(x)}><Settings size={14}/> {x.enabled?'禁用':'启用'}</button><button onClick={()=>x.secret?copyText(x.secret,setMsg):setErr('历史 Key 未保存明文，请重新生成后复制完整 Key')}><Copy size={14}/> 复制完整 Key</button><button onClick={()=>rotateKey(x)}><RefreshCw size={14}/> 重新生成 Key</button><button className="danger" onClick={()=>del(x.id)}><Trash2 size={14}/> 删除</button></RowMenu></td></tr>{editing===x.id&&<tr className="edit-row"><td colSpan="10"><div className="inline-edit"><label className="wide">名称<input value={editLabel} onChange={e=>setEditLabel(e.target.value)} /></label><label>权限<select value={editPermissions[0]||'tts'} onChange={e=>setEditPermissions([e.target.value])}><option value='tts'>tts</option><option value='*'>*</option></select></label><button className="primary" onClick={()=>saveEdit(x.id)}>保存</button><button className="btn" onClick={()=>setEditing(null)}>取消</button></div></td></tr>}</React.Fragment>)}{rows.length===0&&<tr><td colSpan="10" className="empty-cell">暂无 API Key</td></tr>}</tbody></table></div>
  </PageShell>
}


/* ====================== Settings ====================== */
function SettingsPage({ auth, setMsg, setErr }) {
  const [adminToken, setAdminToken] = useState('')
  const [proxyUrl, setProxyUrl] = useState('')
  const [adminTokenSet, setAdminTokenSet] = useState(false)

  async function load() {
    try {
      const data = await (await api('/api/admin/settings', { headers: auth })).json()
      setProxyUrl(data.proxyUrl || '')
      setAdminTokenSet(!!data.adminTokenSet)
    } catch (e) { setErr(e.message) }
  }
  useEffect(() => { load() }, [])

  async function save() {
    try {
      const body = { proxyUrl }
      if (adminToken.trim()) body.adminToken = adminToken.trim()
      await api('/api/admin/settings', { method: 'PATCH', headers: auth, body: JSON.stringify(body) })
      if (adminToken.trim()) {
        localStorage.setItem('adminToken', adminToken.trim())
        setAdminToken('')
        setAdminTokenSet(true)
      }
      setMsg('系统设置已保存')
      await load()
    } catch (e) { setErr(e.message) }
  }

  return <div>
    <div className="section-header mb-4"><h1>系统设置</h1><p>修改后台登录 Token，并配置后端访问上游服务时使用的代理。</p></div>
    <div className="card-panel">
      <div className="card-panel-title">管理员 Token</div>
      <div className="card-panel-desc">留空表示不修改。保存新 Token 后，本浏览器会自动切换到新 Token。</div>
      <label className="field-label">新的 ADMIN_TOKEN</label>
      <input type="password" value={adminToken} onChange={e => setAdminToken(e.target.value)} placeholder={adminTokenSet ? '已设置运行时 Token，留空不修改' : '留空则继续使用环境变量 ADMIN_TOKEN'} />
    </div>
    <div className="card-panel">
      <div className="card-panel-title">出站代理</div>
      <div className="card-panel-desc">用于读取上游 /models 和实际 TTS 调用。支持 http://、https://、socks5://；留空表示直连。</div>
      <label className="field-label">代理地址</label>
      <input value={proxyUrl} onChange={e => setProxyUrl(e.target.value)} placeholder="例如：http://127.0.0.1:7890 或 socks5://host.docker.internal:7890" />
    </div>
    <button className="primary" onClick={save}>保存设置</button>
  </div>
}

/* ====================== Call Logs ====================== */
function CallLogs({ auth, setMsg, setErr }) {
  const [logs, setLogs] = useState([])
  const [total, setTotal] = useState(0)
  const [stats, setStats] = useState(null)
  const [page, setPage] = useState(0)
  const [limit] = useState(20)
  const [filterEndpoint, setFilterEndpoint] = useState('')
  const [filterSuccess, setFilterSuccess] = useState('')
  const [purgeDays, setPurgeDays] = useState(30)

  async function loadLogs() {
    try {
      const params = new URLSearchParams({ limit: String(limit), offset: String(page * limit) })
      if (filterEndpoint) params.set('endpoint', filterEndpoint)
      if (filterSuccess !== '') params.set('success', filterSuccess)
      const data = await (await api(`/api/admin/call-logs?${params}`, { headers: auth })).json()
      setLogs(data.logs || [])
      setTotal(data.total || 0)
    } catch (e) { setErr(e.message) }
  }

  async function loadStats() {
    try { setStats(await (await api('/api/admin/call-logs/stats', { headers: auth })).json()) }
    catch (e) { setErr(e.message) }
  }

  async function purge() {
    if (!confirm(`确认清理 ${purgeDays} 天前的调用记录？此操作不可撤销。`)) return
    try {
      const data = await (await api(`/api/admin/call-logs?olderThanDays=${purgeDays}`, { method: 'DELETE', headers: auth })).json()
      setMsg(`已清理 ${data.deleted} 条记录`); await loadLogs(); await loadStats()
    } catch (e) { setErr(e.message) }
  }

  useEffect(() => { loadLogs(); loadStats() }, [page, filterEndpoint, filterSuccess])

  const totalPages = Math.ceil(total / limit)

  function fmtTime(ts) { if (!ts) return '—'; return new Date(ts).toLocaleString() }
  function fmtBytes(n) { if (n < 1024) return n + ' B'; return (n / 1024).toFixed(1) + ' KB' }
  function fmtDur(ms) { if (ms < 1000) return ms + ' ms'; return (ms / 1000).toFixed(2) + ' s' }

  return (
    <div>
      <div className="section-header mb-4">
        <h1><Activity size={20} style={{ display: 'inline', verticalAlign: 'middle' }} /> 调用记录</h1>
        <p>查看所有 TTS 调用的详细记录和统计</p>
      </div>

      {/* Stats */}
      {stats && (
        <div className="calllog-stats">
          <div className="stat-card"><span className="stat-val">{stats.totalCalls}</span><span className="stat-label">总调用</span></div>
          <div className="stat-card"><span className="stat-val" style={{ color: 'var(--success)' }}>{stats.successCalls}</span><span className="stat-label">成功</span></div>
          <div className="stat-card"><span className="stat-val" style={{ color: 'var(--danger)' }}>{stats.failedCalls}</span><span className="stat-label">失败</span></div>
          <div className="stat-card"><span className="stat-val">{fmtDur(stats.avgDurationMs)}</span><span className="stat-label">平均耗时</span></div>
          <div className="stat-card"><span className="stat-val">{fmtBytes(stats.totalAudioBytes)}</span><span className="stat-label">总音频</span></div>
        </div>
      )}

      {/* Filters */}
      <div className="card-panel" style={{ padding: '12px 16px', marginBottom: 16 }}>
        <div className="row" style={{ margin: 0 }}>
          <select value={filterEndpoint} onChange={e => { setFilterEndpoint(e.target.value); setPage(0) }}>
            <option value="">全部端点</option>
            <option value="/api/tts">/api/tts</option>
            <option value="/v1/audio/speech">/v1/audio/speech</option>
          </select>
          <select value={filterSuccess} onChange={e => { setFilterSuccess(e.target.value); setPage(0) }}>
            <option value="">全部状态</option>
            <option value="true">成功</option>
            <option value="false">失败</option>
          </select>
          <button className="smallbtn" onClick={() => { loadLogs(); loadStats() }}><RefreshCw size={14} /> 刷新</button>
        </div>
      </div>

      {/* Logs table */}
      <div className="calllog-table-wrap">
        <table className="calllog-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>端点</th>
              <th>模型</th>
              <th>音色</th>
              <th>Key / 供应商</th>
              <th>状态</th>
              <th>耗时</th>
              <th>音频</th>
              <th>字符</th>
              <th>调用者</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(l => (
              <tr key={l.requestId} className={l.success ? '' : 'failed'}>
                <td className="mono">{fmtTime(l.timestamp)}</td>
                <td><code>{l.endpoint}</code></td>
                <td>
                  <div className="mono">{l.model || '—'}</div>
                  {l.requestedModel && l.requestedModel !== l.model && <div className="subtle">请求：{l.requestedModel}</div>}
                </td>
                <td>
                  <div>{l.voice || '—'}</div>
                  {(l.voiceKind || l.projectVoiceId || (l.requestedVoice && l.requestedVoice !== l.voice)) && (
                    <div className="subtle">
                      {l.voiceKind && <span>{l.voiceKind}</span>}
                      {l.projectVoiceId && <span> · {l.projectVoiceId}</span>}
                      {l.requestedVoice && l.requestedVoice !== l.voice && <span> · 请求：{l.requestedVoice}</span>}
                    </div>
                  )}
                </td>
                <td className="upstream-cell" title={`${l.upstreamLabel || ''} ${l.upstreamId || ''} ${l.upstreamBaseUrl || ''}`.trim()}>
                  <div className="upstream-label">{l.upstreamLabel || '未知上游'}</div>
                  <div className="upstream-meta">{l.upstreamId || '—'}</div>
                  <div className="upstream-url">{l.upstreamBaseUrl || '—'}</div>
                </td>
                <td>{l.success ? <span style={{ color: 'var(--success)' }}>✓ {l.httpStatus}</span> : <span style={{ color: 'var(--danger)' }}>✗ {l.httpStatus}</span>}</td>
                <td className="mono">{fmtDur(l.durationMs)}</td>
                <td className="mono">{fmtBytes(l.audioBytes)}</td>
                <td className="mono">{l.inputChars}</td>
                <td className="mono">{l.callerKeyId || '—'}</td>
              </tr>
            ))}
            {logs.length === 0 && <tr><td colSpan={10} style={{ textAlign: 'center', color: 'var(--fg-muted)', padding: 24 }}>暂无调用记录</td></tr>}
          </tbody>
        </table>
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="row" style={{ justifyContent: 'center' }}>
          <button className="btn" disabled={page === 0} onClick={() => setPage(page - 1)}>上一页</button>
          <span className="hint">第 {page + 1}/{totalPages} 页 (共 {total} 条)</span>
          <button className="btn" disabled={page >= totalPages - 1} onClick={() => setPage(page + 1)}>下一页</button>
        </div>
      )}

      {/* Error details */}
      {logs.filter(l => l.error).length > 0 && (
        <details className="calllog-errors mt-4">
          <summary>失败详情</summary>
          {logs.filter(l => l.error).map(l => (
            <div key={l.requestId} className="calllog-error-row">
              <code>{l.requestId}</code> <span>{fmtTime(l.timestamp)}</span> <span className="error-text">{l.error}</span>
            </div>
          ))}
        </details>
      )}

      {/* Purge */}
      <details className="calllog-purge mt-4">
        <summary>清理旧记录</summary>
        <div className="row mt-2">
          <input type="number" min="1" max="365" value={purgeDays} onChange={e => setPurgeDays(Number(e.target.value))} style={{ maxWidth: 100 }} />
          <span className="hint">天前的记录</span>
          <button className="btn" onClick={purge} style={{ background: 'var(--danger)', color: '#fff', borderColor: 'var(--danger)' }}>清理</button>
        </div>
      </details>
    </div>
  )
}
