import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--disable-dev-shm-usage','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1440,height:900}}),errors=[];
 page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  window.pluginCalls=[];window.pluginFailure='';window.restartFailure=false;
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:true,version:'150'}}),registerBrowserWindows:()=>{},
   syncWindowDirectories:()=>new Promise(()=>{}),syncBuiltInExtensions:async()=>{window.pluginCalls.push(['unexpected-sync']);return[]},
   startBrowserWindow:async input=>{window.pluginCalls.push(['start',input]);return {ok:true,profileId:input.profileId}},stopBrowserWindow:async id=>window.pluginCalls.push(['stop',id]),
   setWindowPlugins:async input=>{window.pluginCalls.push(['set',input]);if(window.pluginFailure===input.id)throw new Error('simulated apply failure');return {profileId:input.profileId,running:true}},
   restartWindowForPlugins:async input=>{window.pluginCalls.push(['restart',input]);if(window.restartFailure)throw new Error('插件已关联，请手动关闭并重启');return {profileId:input.profileId,running:true}},
   importPlugin:async()=>({extensionId:'cccccccccccccccccccccccccccccccc',name:'Imported test'}),logOperation:async()=>{}};
  if(localStorage.getItem('__windowPluginsFixture'))return;
  localStorage.clear();localStorage.setItem('__windowPluginsFixture','1');localStorage.setItem('one-browser-plugins',JSON.stringify({adguard:true,tampermonkey:true,'google-translate':true}));
  localStorage.setItem('one-browser-environments',JSON.stringify(['A','B','C'].map((name,i)=>({windowId:crypto.randomUUID(),profileId:'profile-'+name,name,state:i<2?'running':'stopped',config:{browser:'chrome'},tag:'新环境'}))));
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);await page.waitForTimeout(160);
 const ids=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments')).map(e=>e.windowId));
 const entries=()=>page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments')));
 await page.locator('#envRows tr').nth(2).locator('[data-action="start"]').click();
 await page.waitForFunction(()=>window.pluginCalls.some(c=>c[0]==='start'));
 assert.deepEqual((await page.evaluate(()=>window.pluginCalls.find(c=>c[0]==='start')[1])).extensions,[],'first start must have no builtins despite old global defaults');
 assert.equal((await page.evaluate(()=>window.pluginCalls)).filter(c=>c[0]==='unexpected-sync').length,0,'no eager plugin download');
 await page.locator('#envRows tr').nth(2).locator('[data-action="stop"]').click();
 await page.locator('[data-page="plugins"]').click();
 const ad=page.locator('[data-plugin="adguard"]'),modal=page.locator('#pluginWindowModal');
 assert.equal(await page.locator('[data-plugin]:checked').count(),0);
 await ad.locator('..').click();await modal.locator('#cancelPluginWindows').click();assert.equal(await ad.isChecked(),false);assert.equal((await page.evaluate(()=>window.pluginCalls)).filter(c=>c[0]==='set').length,0);
 const choose=async names=>{for(let i=0;i<ids.length;i++)await page.locator(`#pluginWindowList input[value="${ids[i]}"]`).setChecked(names.includes(i))};
 await ad.locator('..').click();assert.equal(await page.locator('#pluginWindowList input:checked').count(),2);assert.equal(await page.locator('#pluginRestartWindows').isChecked(),false);
 await choose([0]);await page.locator('#applyPluginWindows').click();await modal.waitFor({state:'hidden'});
 let saved=await entries();assert.deepEqual(saved[0].pluginKeys,['adguard']);assert.deepEqual(saved[1].pluginKeys,[]);assert.deepEqual(saved[2].pluginKeys,[]);
 assert.equal((await page.evaluate(()=>window.pluginCalls)).filter(c=>c[0]==='restart').length,0,'association must not silently restart');
 await page.locator('[data-page="environments"]').click();await page.locator('#envRows tr').nth(1).locator('[data-action="stop"]').click();await page.locator('#envRows tr').nth(1).locator('[data-action="start"]').click();
 const bStart=await page.evaluate(()=>window.pluginCalls.filter(c=>c[0]==='start').at(-1)[1]);assert.equal(bStart.id,ids[1]);assert.deepEqual(bStart.extensions,[],'another window must not inherit enabled global card');
 await page.locator('[data-page="plugins"]').click();await ad.locator('..').click();await choose([0,1]);await page.locator('#pluginRestartWindows').check();await page.locator('#applyPluginWindows').click();await modal.waitFor({state:'hidden'});
 assert.equal((await page.evaluate(()=>window.pluginCalls)).filter(c=>c[0]==='restart').length,1);saved=await entries();assert.deepEqual(saved[1].pluginKeys,['adguard']);
 await ad.locator('..').click();await choose([1]);await page.locator('#applyPluginWindows').click();await modal.waitFor({state:'hidden'});saved=await entries();assert.deepEqual(saved[0].pluginKeys,[]);assert.deepEqual(saved[1].pluginKeys,['adguard']);
 await page.evaluate(id=>window.pluginFailure=id,ids[2]);await page.locator('[data-plugin="tampermonkey"]').locator('..').click();await page.locator('#pluginSelectAll').click();await page.locator('#applyPluginWindows').click();
 await page.waitForFunction(()=>document.getElementById('pluginWindowMessage').textContent.includes('simulated apply failure'));
 saved=await entries();assert(saved[0].pluginKeys.includes('tampermonkey'));assert(saved[1].pluginKeys.includes('tampermonkey'));assert.deepEqual(saved[2].pluginKeys,[],'failure must not commit local association');
 await page.locator('#cancelPluginWindows').click();await page.evaluate(()=>window.pluginFailure='');
 await page.locator('#importPlugin').click();await choose([2]);await page.locator('#applyPluginWindows').click();await modal.waitFor({state:'hidden'});saved=await entries();assert.deepEqual(saved[2].customPluginIDs,['cccccccccccccccccccccccccccccccc']);

 await page.locator('[data-rename-plugin="cccccccccccccccccccccccccccccccc"]').click();
 await page.locator('.plugin-name-input').fill('Translator renamed');await page.locator('.plugin-name-input').press('Enter');
 assert.equal(await page.locator('.imported-plugin-card .plugin-icon').textContent(),'T');
 assert.equal(await page.locator('.imported-plugin-card b').textContent(),'Translator renamed');
 assert.equal(await page.locator('.imported-plugin-card button').filter({hasText:'改名'}).count(),0,'no separate rename button');
 await page.locator('[data-rename-plugin="cccccccccccccccccccccccccccccccc"]').click();
 await page.locator('.plugin-name-input').fill('Blur save');await page.locator('#pageTitle').click();
 assert.equal(await page.locator('.imported-plugin-card .plugin-icon').textContent(),'B');
 await page.locator('[data-rename-plugin="cccccccccccccccccccccccccccccccc"]').focus();
 await page.locator('[data-rename-plugin="cccccccccccccccccccccccccccccccc"]').press('Enter');
 await page.locator('.plugin-name-input').fill('Translator renamed');await page.locator('.plugin-name-input').press('Enter');

 await page.locator('#importPlugin').click();
 assert((await page.locator('#pluginWindowTitle').textContent()).includes('Translator renamed'),'reimport must retain custom display name');
 await page.locator('#cancelPluginWindows').click();

 await page.locator('[data-rename-plugin="cccccccccccccccccccccccccccccccc"]').click();
 await page.locator('.plugin-name-input').fill('Discarded');await page.locator('.plugin-name-input').press('Escape');
 assert.equal(await page.locator('.imported-plugin-card b').textContent(),'Translator renamed');
 await page.locator('[data-imported-toggle="cccccccccccccccccccccccccccccccc"]').locator('..').click();
 assert((await page.locator('#pluginWindowTitle').textContent()).startsWith('关联窗口：'));await choose([]);
 await page.locator('#applyPluginWindows').click();await modal.waitFor({state:'hidden'});
 saved=await entries();assert.deepEqual(saved[2].customPluginIDs,[]);
 assert.deepEqual(await page.evaluate(()=>window.pluginCalls.filter(c=>c[0]==='set').at(-1)[1].removedExtensionIds),['cccccccccccccccccccccccccccccccc']);

 await page.evaluate(()=>window.restartFailure=true);await page.locator('[data-plugin="google-translate"]').locator('..').click();await choose([0]);await page.locator('#pluginRestartWindows').check();await page.locator('#applyPluginWindows').click();
 await page.waitForFunction(()=>document.getElementById('pluginWindowMessage').textContent.includes('请手动关闭并重启'));
 saved=await entries();assert(saved[0].pluginKeys.includes('google-translate'),'saved association must remain after graceful restart failure');await page.locator('#cancelPluginWindows').click();
 await page.reload();await page.locator('[data-page="plugins"]').click();assert(await page.locator('[data-plugin="adguard"]').isChecked());assert((await page.locator('#importedPluginCards').textContent()).includes('Translator renamed'));
 await page.setViewportSize({width:900,height:680});await ad.locator('..').click();const apply=await page.locator('#applyPluginWindows').boundingBox();assert(apply.y+apply.height<=680,'picker actions must fit small window');
 assert.deepEqual(errors,[]);
 console.log('Window plugins UI passed: default off, no eager download/global wait, opt-in per window, batch/partial failure, explicit restart, import, reload and compact picker');
}finally{await browser.close()}
