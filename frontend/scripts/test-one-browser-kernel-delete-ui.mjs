import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1440,height:900}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  localStorage.clear();window.removes=[];window.coreChanges=[];window.deleteFailure=false;window.pluginDeleteFailure=false;window.pluginDeletes=[];
  const id=crypto.randomUUID(),plugin='cccccccccccccccccccccccccccccccc';let binding='v150';let cores=[{installed:true,coreId:'v150',version:'150.0.7871.186'},{installed:true,coreId:'v148',version:'148.0.7778.215'}];
  localStorage.setItem('one-browser-environments',JSON.stringify([{windowId:id,profileId:'profile-A',name:'A',state:'stopped',pluginKeys:['adguard'],customPluginIDs:[plugin],config:{browser:'chrome',coreId:'v150',country:'de'}}]));
  localStorage.setItem('one-browser-imported-plugins',JSON.stringify([{extensionId:plugin,name:'Test Plugin'}]));
  window.onlineDesktop={getKernelStatus:async()=>({chrome:cores[0]||{installed:false}}),listKernels:async()=>cores,
   getWindowKernelBindings:async()=>[{windowId:id,profileId:'profile-A',coreId:binding}],
   setWindowKernel:async input=>{window.coreChanges.push(input);binding=input.coreId},
   removeKernel:async(key,coreId,windows)=>{window.removes.push({key,coreId,windows});if(window.deleteFailure)throw Error('simulated delete failure');cores=cores.filter(c=>c.coreId!==coreId)},
   deleteImportedPlugin:async id=>{window.pluginDeletes.push(id);if(window.pluginDeleteFailure)throw Error('simulated plugin failure')},
   registerBrowserWindows:()=>{},renameBrowserWindow:async()=>({}),syncWindowDirectories:async()=>[],logOperation:async()=>{}};
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);
 await page.waitForFunction(()=>document.querySelectorAll('.installed-kernel-card').length===2);
 await page.locator('#createBtn').click();assert.equal(await page.locator('#browserVersion').inputValue(),'v150');
 assert.equal(await page.locator('#browserVersion option[value=""]').count(),0,'remove pseudo default option');
 assert.match(await page.locator('#presetCountry').evaluate(e=>e.parentElement.querySelector('.rounded-select-trigger').textContent),/美国/);
 assert.doesNotMatch(await page.locator('#presetCountry').evaluate(e=>e.parentElement.querySelector('.rounded-select-trigger').textContent),/Chromium/);
 await page.locator('#cancelModal').click();await page.locator('[data-page="kernels"]').click();
 const strip=await page.locator('.kernel-secondary-actions').boundingBox(),card=await page.locator('.installed-kernel-card').first().boundingBox();assert(strip.y+strip.height<=card.y,'imports above installed cores');
 await page.locator('[data-delete-core="v150"]').click();assert.equal(await page.locator('#deleteKernelModal').isVisible(),false);assert.match(await page.locator('#toast').innerText(),/编辑窗口/);assert.equal(await page.evaluate(()=>window.removes.length),0);
 await page.locator('[data-page="environments"]').click();await page.evaluate(()=>openEditor(0));
 await page.locator('#browserVersion').evaluate(e=>{e.value='v148';e.dispatchEvent(new Event('change',{bubbles:true}))});await page.locator('#submitEnv').click();
 await page.waitForFunction(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0].config.coreId==='v148');assert.equal(await page.evaluate(()=>window.coreChanges.at(-1).coreId),'v148');
 await page.locator('[data-page="kernels"]').click();await page.locator('[data-delete-core="v150"]').click();await page.locator('#cancelDeleteKernel').click();assert.equal(await page.evaluate(()=>window.removes.length),0,'cancel must not delete');
 await page.locator('[data-delete-core="v150"]').click();await page.evaluate(()=>window.deleteFailure=true);await page.locator('#confirmDeleteKernel').click();
 await page.waitForFunction(()=>document.querySelector('#toast').textContent.includes('simulated delete failure'));assert.equal(await page.locator('.installed-kernel-card').count(),2);
 await page.evaluate(()=>window.deleteFailure=false);await page.locator('#confirmDeleteKernel').click();await page.waitForFunction(()=>document.querySelectorAll('.installed-kernel-card').length===1);
 assert.equal(await page.locator('[data-core-id="v148"]').count(),1);
 await page.locator('[data-page="plugins"]').click();await page.locator('[data-delete-plugin]').click();await page.locator('#cancelDeleteImportedPlugin').click();assert.equal(await page.evaluate(()=>window.pluginDeletes.length),0);
 await page.locator('[data-delete-plugin]').click();await page.evaluate(()=>window.pluginDeleteFailure=true);await page.locator('#confirmDeleteImportedPlugin').click();
 await page.waitForFunction(()=>document.querySelector('#toast').textContent.includes('simulated plugin failure'));assert.equal(await page.locator('.imported-plugin-card').count(),1);assert.equal(await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0].customPluginIDs.length),1);
 await page.evaluate(()=>window.pluginDeleteFailure=false);await page.locator('#confirmDeleteImportedPlugin').click();await page.waitForFunction(()=>document.querySelectorAll('.imported-plugin-card').length===0);
 const entry=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0]);assert.deepEqual(entry.customPluginIDs,[]);assert.deepEqual(entry.pluginKeys,['adguard']);assert.deepEqual(await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-imported-plugins'))),[]);
 assert.deepEqual(errors,[]);console.log('Newest default, country label, top imports, associated core guards, edit persistence, confirmed deletion and plugin failure preservation passed');
}finally{await browser.close()}
