(function(root){
 'use strict';
 const builtinIDs={adguard:'bgnkhhnnamicmpeenaelnjfhikgbkllg',tampermonkey:'dhdgffkkebhmkfjojejmpbldmpobfkfo','google-translate':'aapbdbdomjkkjkaonfhkkikfgjllcleb'};
 const names={adguard:'AdGuard 广告拦截器',tampermonkey:'篡改猴','google-translate':'Google 翻译'};
 function normalize(entries){for(const entry of entries){entry.pluginKeys=[...new Set((Array.isArray(entry.pluginKeys)?entry.pluginKeys:[]).filter(key=>Object.hasOwn(builtinIDs,key)))];entry.customPluginIDs=[...new Set((Array.isArray(entry.customPluginIDs)?entry.customPluginIDs:[]).filter(id=>/^[a-p]{32}$/.test(id)))]}return entries}
 function pluginName(name,id){const value=String(name||'').trim();return value&&!/^__MSG_/i.test(value)?value:'导入插件 '+id.slice(0,6)}
 function keys(entry){return [...(entry?.pluginKeys||[])]}
 function init(ctx){
  const $=id=>document.getElementById(id),modal=$('pluginWindowModal'),list=$('pluginWindowList'),restart=$('pluginRestartWindows'),message=$('pluginWindowMessage');
  let pending=null,busy=false,returnFocus=null,catalog=[],deletingPlugin=null;
  try{catalog=JSON.parse(localStorage.getItem('one-browser-imported-plugins')||'[]').filter(item=>/^[a-p]{32}$/.test(item.extensionId)&&!Object.values(builtinIDs).includes(item.extensionId))}catch(_){}
  normalize(ctx.entries);localStorage.removeItem('one-browser-plugins');
  const linked=(entry,plugin)=>plugin.key?entry.pluginKeys.includes(plugin.key):entry.customPluginIDs.includes(plugin.extensionId);
  function render(){
   normalize(ctx.entries);
   document.querySelectorAll('[data-plugin]').forEach(input=>{const count=ctx.entries.filter(entry=>entry.pluginKeys.includes(input.dataset.plugin)).length;input.checked=count>0;input.disabled=busy;const card=input.closest('.plugin-card');card.style.setProperty('--plugin-hue',String((227+Object.keys(builtinIDs).indexOf(input.dataset.plugin)*137.508)%360));input.setAttribute('aria-label',names[input.dataset.plugin]+'，选择安装窗口');const hint=input.closest('.plugin-card').querySelector('.plugin-assignment');if(hint)hint.textContent=count?'已关联 '+count+' 个窗口':''});
   const imported=$('importedPluginCards');
   imported.innerHTML=catalog.map((plugin,index)=>{const count=ctx.entries.filter(entry=>linked(entry,plugin)).length,name=pluginName(plugin.displayName||plugin.name,plugin.extensionId);return `<article class="plugin-card imported-plugin-card" style="--plugin-hue:${(227+(index+3)*137.508)%360}"><div class="plugin-icon">${ctx.escape(([...name][0]||'E').toUpperCase())}</div><div class="plugin-copy"><div class="plugin-name-row"><b title="点击修改名称：${ctx.escape(name)}" data-rename-plugin="${plugin.extensionId}" role="button" tabindex="0" aria-label="修改插件名称：${ctx.escape(name)}" aria-disabled="${busy}">${ctx.escape(name)}</b></div><small class="plugin-assignment">${count?'已关联 '+count+' 个窗口':''}</small><div class="plugin-card-links"><button type="button" class="plugin-remove-soft" data-delete-plugin="${plugin.extensionId}" ${busy?'disabled':''} aria-label="删除插件：${ctx.escape(name)}">删除</button></div></div><label class="switch plugin-state-toggle"><input type="checkbox" data-imported-toggle="${plugin.extensionId}" aria-label="${ctx.escape(name)}，选择安装窗口" ${count?'checked':''} ${busy?'disabled':''}><span></span></label></article>`}).join('');
  }
  const deleteModal=$('deleteImportedPluginModal');
  function requestDelete(id){if(busy)return;const plugin=catalog.find(item=>item.extensionId===id);if(!plugin)return;const active=ctx.entries.filter(entry=>linked(entry,plugin)&&(entry.starting||entry.state==='running'));if(active.length)return ctx.notify('请先停止使用此插件的窗口：'+active.map(entry=>entry.name).join('、'));deletingPlugin=plugin;$('deleteImportedPluginText').textContent='确定删除“'+pluginName(plugin.displayName||plugin.name,id)+'”？将解除所有窗口关联并移除插件文件，浏览器其他数据保留。';deleteModal.classList.add('open')}
  function closeDelete(){if(busy)return;deleteModal.classList.remove('open');deletingPlugin=null}
  $('cancelDeleteImportedPlugin').onclick=closeDelete;deleteModal.onclick=event=>{if(event.target===deleteModal)closeDelete()};
  $('confirmDeleteImportedPlugin').onclick=async()=>{const plugin=deletingPlugin;if(!plugin||busy)return;const desktop=ctx.desktop();if(!desktop?.deleteImportedPlugin)return ctx.notify('插件删除服务未就绪');busy=true;render();const button=$('confirmDeleteImportedPlugin');button.disabled=true;button.textContent='正在删除';try{await desktop.deleteImportedPlugin(plugin.extensionId);for(const entry of ctx.entries)entry.customPluginIDs=entry.customPluginIDs.filter(id=>id!==plugin.extensionId);catalog=catalog.filter(item=>item.extensionId!==plugin.extensionId);localStorage.setItem('one-browser-imported-plugins',JSON.stringify(catalog));ctx.save();busy=false;closeDelete();render();ctx.notify('插件已删除，窗口关联已解除')}catch(error){ctx.notify(error.message||'删除失败，插件已保留')}finally{busy=false;button.disabled=false;button.textContent='确认删除';render()}};
  function close(){if(busy)return;modal.classList.remove('open');modal.setAttribute('aria-hidden','true');pending=null;returnFocus?.focus()}
  function selected(){return [...list.querySelectorAll('input:checked:not(:disabled)')].map(input=>ctx.entries.find(entry=>entry.windowId===input.value)).filter(Boolean)}
  function selectionChanged(){const targets=selected(),running=targets.some(entry=>entry.state==='running');restart.disabled=!running;if(!running)restart.checked=false;$('applyPluginWindows').disabled=busy||(!pending?.edit&&!targets.length);$('pluginRestartLabel').classList.toggle('is-disabled',!running)}
  function open(plugin,enable=true,edit=false){
   if(busy)return;
   if(!ctx.entries.length){pending=null;modal.classList.remove('open');modal.setAttribute('aria-hidden','true');if(document.activeElement?.closest('.plugin-card'))document.activeElement.blur();ctx.notify('请先创建浏览器窗口，再关联插件');render();return}
   pending={...plugin,enable,edit};returnFocus=document.activeElement;message.textContent='';restart.checked=false;
   $('pluginWindowTitle').textContent=(edit?'关联窗口：':enable?'关联插件：':'停用插件：')+pluginName(plugin.displayName||plugin.name,plugin.extensionId||'');
   const running=ctx.entries.filter(entry=>entry.state==='running'&&!entry.starting);
   list.innerHTML=ctx.entries.map(entry=>{const hasLinked=ctx.entries.some(item=>linked(item,plugin)),checked=edit?(hasLinked?linked(entry,plugin):running.length?running.includes(entry):ctx.entries.length===1):enable?(running.length?running.includes(entry):ctx.entries.length===1):linked(entry,plugin);return `<label class="plugin-window-option"><input type="checkbox" value="${ctx.escape(entry.windowId)}" ${checked?'checked':''} ${entry.starting?'disabled':''}><span><b>${ctx.escape(entry.name)}</b><small>${entry.starting?'启动中，请稍后':entry.state==='running'?'运行中':'已关闭'}${linked(entry,plugin)?' · 已关联':''}</small></span></label>`}).join('');
   $('applyPluginWindows').textContent=edit?'保存关联':enable?'确认关联':'确认停用';selectionChanged();modal.classList.add('open');modal.setAttribute('aria-hidden','false');$('closePluginWindows').focus();
  }
  async function apply(){
   const chosen=selected(),selectedIDs=new Set(chosen.map(entry=>entry.windowId)),targets=pending?.edit?ctx.entries.filter(entry=>!entry.starting&&(selectedIDs.has(entry.windowId)||linked(entry,pending))):chosen;if(busy||!pending||(!pending.edit&&!targets.length))return;
   const plugin={...pending},shouldRestart=restart.checked,errors=[];busy=true;render();modal.querySelectorAll('button,input').forEach(input=>input.disabled=true);
   let saved=0,restarted=0;
   for(const [index,entry] of targets.entries()){
    message.textContent=`正在处理 ${index+1}/${targets.length}：${entry.name}`;
    const enable=plugin.edit?selectedIDs.has(entry.windowId):plugin.enable,nextKeys=keys(entry),nextCustom=[...entry.customPluginIDs],values=plugin.key?nextKeys:nextCustom,value=plugin.key||plugin.extensionId;
    if(enable&&!values.includes(value))values.push(value);else if(!enable&&values.includes(value))values.splice(values.indexOf(value),1);
    if(plugin.edit&&JSON.stringify(nextKeys)===JSON.stringify(keys(entry))&&JSON.stringify(nextCustom)===JSON.stringify(entry.customPluginIDs))continue;
    try{
     if(entry.starting||!ctx.entries.includes(entry))throw new Error('窗口状态已变化，请重新选择');
     const desktop=ctx.desktop();if(!desktop?.setWindowPlugins)throw new Error('窗口插件服务未就绪');
     const result=await desktop.setWindowPlugins({id:entry.windowId,profileId:entry.profileId||'',keys:nextKeys,extensionIds:nextCustom,removedExtensionIds:!plugin.key&&!enable?[plugin.extensionId]:[]});
     entry.pluginKeys=nextKeys;entry.customPluginIDs=nextCustom;saved++;ctx.save();
     if(shouldRestart&&(result?.running||entry.state==='running')){
      if(!desktop.restartWindowForPlugins)throw new Error('插件已关联，请手动关闭并重启该窗口');
      message.textContent='正在正常关闭并重新启动：'+entry.name;
      const started=await desktop.restartWindowForPlugins({id:entry.windowId,profileId:result?.profileId||entry.profileId});
      if(!started?.running)throw new Error('插件已关联，窗口未能重新启动，请手动启动');
      entry.state='running';entry.profileId=started.profileId||entry.profileId;restarted++;ctx.save();
     }
    }catch(error){errors.push(entry.name+'：'+(error.message||String(error)))}
   }
   busy=false;modal.querySelectorAll('button,input').forEach(input=>input.disabled=false);list.querySelectorAll('input').forEach(input=>input.disabled=Boolean(ctx.entries.find(entry=>entry.windowId===input.value)?.starting));render();selectionChanged();
   if(errors.length){message.textContent=`已保存 ${saved} 个窗口的关联；`+errors.join('；');ctx.notify('部分窗口处理失败，请查看弹窗提示');return}
   close();if(plugin.edit)return ctx.notify('插件窗口关联已保存'+(restarted?'，已重启 '+restarted+' 个窗口':'，运行中的窗口下次启动生效'));ctx.notify(`已${plugin.enable?'关联':'停用'} ${saved} 个窗口${restarted?'，已重启 '+restarted+' 个窗口':'；运行中的窗口下次启动生效'}`);
  }
  document.querySelectorAll('[data-plugin]').forEach(input=>{input.onchange=()=>{render();open({key:input.dataset.plugin,name:names[input.dataset.plugin]},true,true)}});
  $('importedPluginCards').onclick=event=>{const remove=event.target.closest('[data-delete-plugin]');if(remove){requestDelete(remove.dataset.deletePlugin);return}const rename=event.target.closest('[data-rename-plugin]');if(rename){
   if(busy)return;const plugin=catalog.find(item=>item.extensionId===rename.dataset.renamePlugin);if(!plugin)return;
   const row=rename.closest('.plugin-name-row'),input=document.createElement('input');input.className='plugin-name-input';input.maxLength=80;input.value=pluginName(plugin.displayName||plugin.name,plugin.extensionId);input.setAttribute('aria-label','插件名称');
   let settled=false;row.replaceChildren(input);input.focus();input.select();
   const commit=(blur=false)=>{if(settled)return;const value=input.value.trim();if(!value){ctx.notify('请输入插件名称');if(blur){settled=true;render()}else input.focus();return}settled=true;plugin.displayName=[...value].slice(0,80).join('');localStorage.setItem('one-browser-imported-plugins',JSON.stringify(catalog));render()};
   input.onblur=()=>commit(true);input.onkeydown=e=>{if(e.isComposing)return;if(e.key==='Enter'){e.preventDefault();commit()}else if(e.key==='Escape'){e.preventDefault();settled=true;render()}};return;
  }};
  $('importedPluginCards').onkeydown=event=>{if(event.target.matches('[data-rename-plugin]')&&(event.key==='Enter'||event.key===' ')){event.preventDefault();event.target.click()}};
  $('importedPluginCards').onchange=event=>{const input=event.target.closest('[data-imported-toggle]');if(!input)return;const plugin=catalog.find(item=>item.extensionId===input.dataset.importedToggle);render();if(plugin)open(plugin,true,true)};
  $('importPlugin').onclick=async()=>{
   if(busy)return;const button=$('importPlugin');button.disabled=true;
   try{const desktop=ctx.desktop();if(!desktop?.importPlugin)throw new Error('扩展导入服务未就绪');const result=await desktop.importPlugin();if(result?.canceled)return;if(!/^[a-p]{32}$/.test(result?.extensionId))throw new Error('未取得插件ID，请重试');const key=Object.keys(builtinIDs).find(key=>builtinIDs[key]===result.extensionId);if(key){open({key,name:names[key]});return}const plugin={extensionId:result.extensionId,name:pluginName(result.name,result.extensionId)};const existing=catalog.findIndex(item=>item.extensionId===plugin.extensionId);if(existing>=0)Object.assign(catalog[existing],plugin);else catalog.push(plugin);localStorage.setItem('one-browser-imported-plugins',JSON.stringify(catalog));render();open(catalog.find(item=>item.extensionId===plugin.extensionId))}catch(error){ctx.notify(error.message||'导入扩展失败')}finally{button.disabled=false}
  };
  $('closePluginWindows').onclick=$('cancelPluginWindows').onclick=close;$('applyPluginWindows').onclick=apply;
  $('pluginSelectAll').onclick=()=>{list.querySelectorAll('input:not(:disabled)').forEach(input=>input.checked=true);selectionChanged()};
  $('pluginSelectRunning').onclick=()=>{list.querySelectorAll('input:not(:disabled)').forEach(input=>input.checked=ctx.entries.find(entry=>entry.windowId===input.value)?.state==='running');selectionChanged()};
  list.onchange=selectionChanged;modal.onclick=event=>{if(event.target===modal)close()};
  modal.onkeydown=event=>{if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();close()}};
  render();return {render,keys};
 }
 root.OneBrowserWindowPlugins={init,normalize,keys,builtinIDs};
})(typeof window!=='undefined'?window:globalThis);
