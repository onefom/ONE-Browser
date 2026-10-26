(function(root){
 'use strict';
 function createController(desktop){
  const modal=document.createElement('div');modal.id='ipChangeModal';modal.className='modal-backdrop';modal.setAttribute('aria-hidden','true');
  modal.innerHTML='<section class="modal ip-change-dialog" role="dialog" aria-modal="true" aria-labelledby="ipChangeTitle" aria-describedby="ipChangeHint"><div class="modal-head"><div><h2 id="ipChangeTitle">系统代理 IP 已变更</h2><p id="ipChangeHint">已确认网络出口变化，请选择如何处理。</p></div><button type="button" class="ip-change-dismiss" aria-label="继续使用并关闭提醒">×</button></div><div class="ip-change-body"><div class="ip-change-comparison"><div class="ip-change-address"><p class="ip-change-label">变更前</p><b data-ip-before></b><small data-country-before></small></div><div class="ip-change-arrow" aria-hidden="true">→</div><div class="ip-change-address is-new"><p class="ip-change-label">变更后</p><b data-ip-after></b><small data-country-after></small></div></div><p class="ip-change-label" data-ip-count></p><div class="ip-change-windows"></div><p class="ip-change-note">继续使用将保留当前窗口；关闭仅作用于以上仍在运行的窗口。关闭前请留意尚未提交的内容。</p><p data-ip-error role="status"></p></div><div class="modal-actions"><button type="button" class="ghost" data-ip-continue>继续使用</button><button type="button" class="primary" data-ip-close>关闭受影响窗口</button></div></section>';
  document.body.appendChild(modal);let current=null,busy=false,focusBefore=null;const $=selector=>modal.querySelector(selector);
  function open(change){
   if(!change?.id||!Array.isArray(change.windows)||!change.windows.length)return;
   if(current?.id===change.id)return;
   if(!current)focusBefore=document.activeElement;current=change;
   $('[data-ip-before]').textContent=change.before?.ip||'未知';$('[data-ip-after]').textContent=change.after?.ip||'未知';$('[data-country-before]').textContent=change.before?.country||'未知地区';$('[data-country-after]').textContent=change.after?.country||'未知地区';
   $('[data-ip-count]').textContent='受影响窗口 · '+change.windows.length+' 个';const list=$('.ip-change-windows');list.replaceChildren(...change.windows.map(item=>{const span=document.createElement('span');span.textContent=item.name||'未命名窗口';return span}));
   $('[data-ip-error]').textContent='';modal.classList.add('open');modal.setAttribute('aria-hidden','false');$('[data-ip-continue]').focus();
  }
  async function resolve(close){if(!current||busy)return;busy=true;modal.querySelectorAll('button').forEach(button=>button.disabled=true);$('[data-ip-error]').textContent='';try{await desktop.resolveIPChange(current.id,close);current=null;modal.classList.remove('open');modal.setAttribute('aria-hidden','true');if(focusBefore?.isConnected)focusBefore.focus()}catch(error){$('[data-ip-error]').textContent=error.message||'操作失败，请重试'}finally{busy=false;modal.querySelectorAll('button').forEach(button=>button.disabled=false)}}
  $('[data-ip-continue]').onclick=$('.ip-change-dismiss').onclick=()=>resolve(false);$('[data-ip-close]').onclick=()=>resolve(true);
  modal.addEventListener('keydown',event=>{if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();resolve(false)}if(event.key==='Tab'){const buttons=[...modal.querySelectorAll('button:not(:disabled)')];if(!buttons.length){event.preventDefault();return}const first=buttons[0],last=buttons.at(-1);if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus()}else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}}});
  return {open,element:modal};
 }
 root.OneBrowserIPChange={createController};
 const desktop=root.onlineDesktop;if(!desktop?.startIPMonitor)return;
 const controller=createController(desktop);desktop.onIPChange?.(change=>controller.open(change));
 // Pending state is native, so minimising/reloading the manager cannot lose an alert.
 let refreshing=false,lastFailure=0;
 async function refresh(){if(refreshing)return;refreshing=true;try{await desktop.startIPMonitor();const change=await desktop.getIPChange();if(change)controller.open(change)}catch(error){if(Date.now()-lastFailure>60000){lastFailure=Date.now();desktop.logOperation?.('warn','IPMonitor',false,'IP 提醒服务连接失败，将自动重试')?.catch?.(()=>{})}}finally{refreshing=false}}
 refresh();setInterval(refresh,60000);root.addEventListener('focus',refresh);document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
})(window);
