package backend

// compactOneBrowserWorkspacePage mirrors the concise fingerprint summary used
// by the window configuration screen. Values are rendered from the independent
// profile that owns this workspace file; network values are detected inside the
// launched browser so they follow that profile's proxy route.
const compactOneBrowserWorkspacePage = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>工作台 · {{.Name}}</title><style>
:root{color-scheme:light}*{box-sizing:border-box}body{margin:0;min-height:100vh;padding:30px;font-family:"PingFang SC","Microsoft YaHei","Segoe UI",sans-serif;background:linear-gradient(145deg,#f2f5fc,#eaf0fb);color:#20263d}main{max-width:920px;margin:auto}.top{display:flex;justify-content:space-between;align-items:end;margin-bottom:18px}.top h1{margin:0;font-size:22px;font-weight:600}.top p{margin:6px 0 0;color:#858da4;font-size:12px}.badge{padding:6px 11px;border-radius:999px;background:#e7f7f1;color:#35a37d;font-size:11px}.panel{padding:8px 18px;border-radius:15px;background:rgba(255,255,255,.9);box-shadow:0 9px 28px rgba(66,81,120,.07)}.facts{display:grid;grid-template-columns:1fr 1fr;column-gap:30px}.fact{display:grid;grid-template-columns:112px minmax(0,1fr);align-items:start;gap:12px;min-width:0;padding:11px 2px;border-bottom:1px solid #edf0f6}.fact.wide{grid-column:1/-1}.label{color:#788299;font-size:11px;line-height:1.55}.value{font-size:12px;font-weight:550;line-height:1.55;overflow-wrap:anywhere}.network{color:#5e59d9}.footer{margin:14px 2px 0;color:#8991a5;font-size:11px}@media(min-width:681px){.fact:nth-last-child(-n+2){border-bottom:0}}@media(max-width:680px){body{padding:20px}.facts{grid-template-columns:1fr}.fact{grid-template-columns:104px minmax(0,1fr)}.fact:last-child{border-bottom:0}.top{align-items:start;gap:12px}}
</style></head><body><main><div class="top"><div><h1>{{.Name}}</h1><p>当前窗口的独立浏览器配置</p></div><span class="badge">独立工作区</span></div>
<section class="panel facts">
<div class="fact"><span class="label">操作系统</span><b class="value">{{.OS}}</b></div>
<div class="fact"><span class="label">内核类型</span><b class="value">Chrome</b></div>
<div class="fact wide"><span class="label">User Agent</span><b class="value">{{.UserAgent}}</b></div>
<div class="fact"><span class="label">语言</span><b class="value">自定义 · {{.Language}}</b></div>
<div class="fact"><span class="label">时区</span><b class="value">基于 IP 匹配 · {{.Timezone}}</b></div>
<div class="fact"><span class="label">地理位置提示</span><b class="value">允许</b></div>
<div class="fact"><span class="label">地理位置</span><b class="value network" id="networkLocation">基于 IP 匹配 · 正在检测…</b></div>
<div class="fact"><span class="label">分辨率</span><b class="value">{{.WindowSize}}</b></div>
<div class="fact"><span class="label">字体指纹</span><b class="value">跟随系统</b></div>
<div class="fact"><span class="label">WebRTC</span><b class="value">禁止</b></div>
<div class="fact"><span class="label">代理</span><b class="value">{{.Proxy}}</b></div>
<div class="fact"><span class="label">账号</span><b class="value">{{.Account}}</b></div>
<div class="fact"><span class="label">内核版本</span><b class="value">{{.Version}}</b></div>
<div class="fact wide"><span class="label">出口 IP</span><b class="value network" id="publicIp">正在检测…</b></div>
</section><p class="footer">Google 已在相邻标签页打开。出口信息由此独立窗口的网络实时检测。</p></main>
<script>Promise.allSettled([fetch('https://api.ipify.org?format=json').then(r=>r.json()).then(v=>publicIp.textContent=v.ip||'检测失败'),fetch('https://ipwho.is/').then(r=>r.json()).then(v=>networkLocation.textContent='基于 IP 匹配 · '+([v.country,v.city].filter(Boolean).join(' · ')||'检测失败'))]).then(results=>results.forEach((r,i)=>{if(r.status==='rejected')(i?networkLocation:publicIp).textContent=i?'基于 IP 匹配 · 检测失败':'检测失败'}));</script>
</body></html>`
