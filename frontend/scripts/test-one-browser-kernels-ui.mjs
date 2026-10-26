import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1280,height:800}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  localStorage.clear();window.checks=0;window.imports=0;window.launches=[];
  const cores=[{installed:true,coreId:'v150',version:'150.0.7871.186'}];
  window.onlineDesktop={getKernelStatus:async()=>({chrome:cores[0]}),listKernels:async()=>cores,
   getLatestKernelRelease:async()=>{window.checks++;return {version:'150.0.7871.186'}},
   importKernelDirectory:async()=>{window.imports++;cores.push({installed:true,coreId:'v148',version:'148.0.7778.215'});return {installed:true,coreId:'v148'}},
   registerBrowserWindows:()=>{},renameBrowserWindow:async()=>({}),syncWindowDirectories:async()=>[],logOperation:async()=>{},
   startBrowserWindow:async input=>{window.launches.push(input);return {ok:true,profileId:'p148'}}};
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);
 await page.locator('[data-page="kernels"]').click();await page.locator('#importKernelDirectory').click();
 await page.waitForFunction(()=>document.querySelector('#kernelInstalledVersions').textContent.includes('148.0.7778.215'));
 assert.equal(await page.locator('#kernelInstalledVersions .installed-kernel-card').count(),2);
 assert.equal(await page.locator('.kernel-card').evaluate(e=>getComputedStyle(e).backdropFilter),'none');
 await page.locator('[data-page="environments"]').click();await page.locator('#createBtn').click();
 await page.locator('#envName').fill('148 test');
 await page.locator('#browserVersion').evaluate(e=>{e.value='v148';e.dispatchEvent(new Event('change',{bubbles:true}))});
 await page.locator('#submitEnv').click();
 let entry=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0]);assert.equal(entry.config.coreId,'v148');
 await page.locator('#envRows [data-action="start"]').first().click();
 await page.waitForFunction(()=>window.launches.length===1);assert.equal(await page.evaluate(()=>window.launches[0].config.coreId),'v148');
 await page.evaluate(()=>openEditor(0));assert.equal(await page.locator('#browserVersion').inputValue(),'v148');await page.locator('#cancelModal').click();
 await page.waitForTimeout(1200);assert.equal(await page.evaluate(()=>window.checks),1,'initial weekly check');
 await page.locator('#coreUpdateBtn').click();await page.locator('#upgradeCore').waitFor({state:'visible'});assert.equal(await page.locator('#upgradeCore').textContent(),'检查更新');await page.locator('#upgradeCore').click();
 assert.equal(await page.evaluate(()=>window.checks),2,'manual check must query even when latest');
 await page.evaluate(()=>weeklyKernelCheck());assert.equal(await page.evaluate(()=>window.checks),2,'no extra weekly request');
 await page.evaluate(()=>{window.onlineDesktop.getLatestKernelRelease=async()=>{throw Error('offline')}});
 await page.locator('#upgradeCore').click();assert.match(await page.locator('#toast').innerText(),/检查更新失败/);
 assert.deepEqual(errors,[]);console.log('Kernel import/list, window selection/save/edit/launch, weekly and manual update checks passed');
}finally{await browser.close()}
