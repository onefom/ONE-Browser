package backend

// compactOneBrowserWorkspacePage is written inside the profile's portable
// data/workspaces directory. Network values are queried by the launched
// browser, so they follow that profile's real proxy route.
const compactOneBrowserWorkspacePage = `<!doctype html>
<html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>工作台 · {{.Name}}</title><style>
:root{color-scheme:light;--ink:#20263d;--muted:#8991a8;--line:#e8ecf5;--violet:#6f63e9;--cyan:#63bddb}*{box-sizing:border-box}body{margin:0;min-height:100vh;padding:22px;font-family:"PingFang SC","Microsoft YaHei","Segoe UI",sans-serif;background:radial-gradient(circle at 15% 6%,rgba(112,192,229,.18),transparent 28%),linear-gradient(145deg,#f3f6fd,#eaf0fb);color:var(--ink)}main{max-width:940px;margin:0 auto}.hero{display:grid;grid-template-columns:minmax(0,1fr) auto;align-items:center;gap:18px;margin-bottom:13px}.identity{display:flex;align-items:center;gap:12px}.logo{display:grid;width:36px;height:36px;place-items:center;border-radius:13px;background:linear-gradient(135deg,var(--cyan),var(--violet) 58%,#d46fd4);color:#fff;font-size:16px;font-weight:700;box-shadow:0 8px 20px rgba(103,96,218,.22)}h1{margin:0;font-size:20px;font-weight:650;line-height:1.25}.subtitle{margin:4px 0 0;color:var(--muted);font-size:10px}.badge{padding:6px 10px;border-radius:999px;background:#e8f8f2;color:#329d77;font-size:10px;white-space:nowrap}.summary{display:grid;grid-template-columns:minmax(0,1.35fr) repeat(3,minmax(120px,.72fr));gap:9px;margin-bottom:9px}.card,.panel{border:1px solid rgba(255,255,255,.9);background:rgba(255,255,255,.82);box-shadow:0 8px 24px rgba(67,82,121,.06);backdrop-filter:blur(14px)}.card{min-width:0;padding:12px 14px;border-radius:13px}.card span{display:block;margin-bottom:6px;color:var(--muted);font-size:9px}.card b{display:block;font-size:11px;font-weight:600;line-height:1.45;overflow-wrap:anywhere}.card.network b{color:#5550cf}.network-line{display:grid;grid-template-columns:minmax(82px,.65fr) minmax(0,1.35fr);gap:12px}.network-line b+b{padding-left:12px;border-left:1px solid var(--line)}.panel{padding:5px 15px;border-radius:15px}.facts{display:grid;grid-template-columns:1fr 1fr;column-gap:28px}.fact{display:grid;grid-template-columns:104px minmax(0,1fr);gap:10px;min-width:0;padding:9px 1px;border-bottom:1px solid var(--line)}.fact.wide{grid-column:1/-1}.label{color:#7d869d;font-size:9px;line-height:1.55}.value{font-size:10px;font-weight:550;line-height:1.55;overflow-wrap:anywhere}.footer{display:flex;justify-content:space-between;gap:18px;margin:10px 2px 0;color:#9299aa;font-size:9px}.footer code{font:inherit;color:#7772b8}@media(min-width:681px){.fact:nth-last-child(-n+2){border-bottom:0}}@media(max-width:760px){body{padding:16px}.summary{grid-template-columns:1fr 1fr}.summary .network{grid-column:1/-1}.facts{grid-template-columns:1fr}.fact.wide{grid-column:auto}.fact{grid-template-columns:96px minmax(0,1fr)}.fact:last-child{border-bottom:0}.footer{display:block}.footer span{display:block;margin-top:4px}}
</style></head><body><main>
<header class="hero"><div class="identity"><span class="logo">O</span><div><h1>{{.Name}}</h1><p class="subtitle">当前窗口的独立配置与实时网络状态</p></div></div><span class="badge">安全隔离运行中</span></header>
<section class="summary">
  <article class="card network"><span>出口 IP / 位置</span><div class="network-line"><b id="publicIp">正在检测 IP…</b><b id="networkLocation">正在检测位置…</b></div></article>
  <article class="card"><span>代理</span><b>{{.Proxy}}</b></article>
  <article class="card"><span>账号</span><b>{{.Account}}</b></article>
  <article class="card"><span>扩展</span><b>{{.Extensions}}</b></article>
</section>
<section class="panel facts">
  <div class="fact"><span class="label">操作系统</span><b class="value">{{.OS}}</b></div><div class="fact"><span class="label">内核类型</span><b class="value">Fingerprint Chromium</b></div>
  <div class="fact"><span class="label">语言</span><b class="value">自定义 · {{.Language}}</b></div><div class="fact"><span class="label">时区</span><b class="value">基于 IP 匹配 · {{.Timezone}}</b></div>
  <div class="fact"><span class="label">窗口尺寸</span><b class="value">{{.WindowSize}}</b></div><div class="fact"><span class="label">内核版本</span><b class="value">{{.Version}}</b></div>
  <div class="fact"><span class="label">地理位置提示</span><b class="value">允许 · 基于 IP 匹配</b></div><div class="fact"><span class="label">WebRTC</span><b class="value">禁止</b></div>
  <div class="fact"><span class="label">字体指纹 / 画布 / WebGL</span><b class="value">跟随系统一致性策略</b></div><div class="fact"><span class="label">数据隔离</span><b class="value">独立 Cookie、缓存与本地存储</b></div>
  <div class="fact wide"><span class="label">User Agent</span><b class="value">{{.UserAgent}}</b></div>
</section>
<p class="footer"><span>Google 已在相邻标签页打开，网络信息由当前独立窗口实时检测。</span><code>工作区 {{.WorkspaceID}}</code></p>
</main><script>
const setFallback=(node,text)=>{node.textContent=text};Promise.allSettled([fetch('https://api.ipify.org?format=json').then(r=>r.json()).then(v=>publicIp.textContent=v.ip||'检测失败'),fetch('https://ipwho.is/').then(r=>r.json()).then(v=>networkLocation.textContent=[v.country,v.city].filter(Boolean).join(' · ')||'检测失败')]).then(results=>results.forEach((r,i)=>{if(r.status==='rejected')setFallback(i?networkLocation:publicIp,'检测失败')}));
</script></body></html>`
