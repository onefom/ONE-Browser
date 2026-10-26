import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1184,height:740}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  localStorage.clear();window.started=[];window.changed=[];
  localStorage.setItem('one-browser-environments',JSON.stringify([{windowId:crypto.randomUUID(),profileId:'fox-profile',name:'Fox',state:'stopped',config:{browser:'chrome',coreId:'fox',version:'158.0',country:'us'}}]));
  localStorage.setItem('one-browser-imported-plugins',JSON.stringify([{extensionId:'cccccccccccccccccccccccccccccccc',name:'Test Plugin'}]));
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:true,coreId:'chrome',version:'150.0'}}),listKernels:async()=>[{installed:true,coreId:'fox',family:'camoufox',version:'158.0'},{installed:true,coreId:'chrome',family:'chromium',version:'150.0'}],registerBrowserWindows:()=>{},syncWindowDirectories:async()=>[],logOperation:async()=>{},renameBrowserWindow:async()=>({}),setWindowKernel:async input=>window.changed.push(input),startBrowserWindow:async input=>{window.started.push(input);return{ok:true,profileId:'fox-profile'}},listImportedPlugins:async()=>[]};
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);
 await page.waitForFunction(()=>document.querySelector('.window-kernel-icon')?.dataset.kernelFamily==='camoufox');
 assert.equal(await page.locator('.launch .window-kernel-icon').count(),1,'logo must be inside the button');
 await page.locator('#createBtn').click();assert.equal(await page.locator('#browserVersion').inputValue(),'chrome','Chrome remains the default when both families are installed');await page.locator('#cancelModal').click();
 await page.evaluate(()=>openEditor(0));assert.equal(await page.locator('#browserVersion').inputValue(),'fox');assert.match(await page.locator('#presetTitle').textContent(),/Camoufox/);await page.locator('#submitEnv').click();
 assert.equal(await page.evaluate(()=>window.changed.at(-1).coreId),'fox');
 await page.locator('.launch').click();await page.waitForFunction(()=>window.started.length===1);assert.equal(await page.evaluate(()=>window.started[0].config.coreId),'fox');
 if(process.env.ONE_BROWSER_SCREENSHOT)await page.screenshot({path:process.env.ONE_BROWSER_SCREENSHOT});
 await page.locator('[data-page="plugins"]').click();
 const buttons=await page.locator('.plugin-card-links').first().evaluate(el=>[...el.children].map(e=>{const r=e.getBoundingClientRect();return{y:r.y,h:r.height}}));assert.equal(buttons.length,2);assert.equal(buttons[0].y,buttons[1].y);assert.equal(buttons[0].h,buttons[1].h);
 assert.deepEqual(errors,[]);console.log('Camoufox family labels, explicit core choice, Chromium default, logo inside launch, start payload and plugin button alignment passed');
}finally{await browser.close()}
