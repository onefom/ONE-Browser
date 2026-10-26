import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1440,height:900}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{
  localStorage.clear();window.calls=[];window.restartError=false;window.actualRunning=true;window.resolveRestart=null;
  localStorage.setItem('one-browser-environments',JSON.stringify(Array.from({length:12},(_,i)=>({windowId:crypto.randomUUID(),profileId:'profile-'+i,name:'Window '+(i+1),state:i===0?'running':'stopped',tag:'Default',lastOpened:'2026/10/23 11:10:20',config:{browser:'chrome',coreId:'v150',version:'150.0.7871.186'}}))));
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:true,version:'150.0.7871.186'}}),listKernels:async()=>[{installed:true,coreId:'v150',version:'150.0.7871.186'}],
   registerBrowserWindows:()=>{},syncWindowDirectories:async()=>[],logOperation:async()=>{},renameBrowserWindow:async()=>({}),setWindowKernel:async()=>{},deleteBrowserWindow:async input=>window.calls.push(['delete',input]),
   startBrowserWindow:async input=>{window.calls.push(['start',input]);return {ok:true,profileId:input.profileId}},stopBrowserWindow:async id=>window.calls.push(['stop',id]),
   restartBrowserWindow:async input=>{window.calls.push(['restart',input]);await new Promise(resolve=>window.resolveRestart=resolve);if(window.restartError)throw Error('simulated restart failure');return {running:true,profileId:input.profileId}},
   getBrowserWindowState:async()=>window.actualRunning};
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);await page.waitForTimeout(100);
 const gear=()=>page.locator('#envRows tr').first().locator('[data-action="menu"]');
 await gear().click();assert.deepEqual(await page.locator('#rowMenu button').allTextContents(),['重启窗口','编辑窗口','复制窗口','删除窗口']);
 await page.locator('[data-menu-action="restart"] svg').click();await page.waitForFunction(()=>window.resolveRestart);
 assert.equal(await page.locator('#envRows tr').first().locator('.launch').isDisabled(),true);await page.evaluate(()=>window.resolveRestart());
 await page.waitForFunction(()=>document.querySelector('#envRows tr .launch .launch-label').textContent==='停止');
 assert.equal((await page.evaluate(()=>window.calls.filter(c=>c[0]==='restart'))).length,1);
 let entries=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments')));assert.equal(entries[0].profileId,'profile-0');assert.equal(entries[0].config.coreId,'v150');
 await page.evaluate(()=>{window.restartError=true;window.resolveRestart=null});await gear().click();await page.locator('[data-menu-action="restart"]').click();await page.waitForFunction(()=>window.resolveRestart);await page.evaluate(()=>window.resolveRestart());
 await page.waitForFunction(()=>document.querySelector('#toast').textContent.includes('simulated restart failure'));assert.equal(await page.locator('#envRows tr').first().locator('.launch .launch-label').textContent(),'停止');
 await gear().click();await page.locator('[data-menu-action="edit"] svg').click();await page.locator('#modal.open').waitFor();assert.equal(await page.locator('#browserVersion').inputValue(),'v150');await page.locator('#cancelModal').click();
 await gear().click();await page.locator('[data-menu-action="clone"] svg').click();entries=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments')));assert.equal(entries.length,13);assert.equal(entries[1].state,'stopped');assert(!entries[1].restarting);assert.notEqual(entries[0].windowId,entries[1].windowId);
 await page.locator('#envRows tr').nth(1).locator('[data-action="menu"]').click();assert.equal(await page.locator('[data-menu-action="restart"]').isDisabled(),true);await page.locator('[data-menu-action="delete"] svg').click();await page.locator('#confirmDeleteWindow').click();await page.waitForFunction(()=>document.querySelectorAll('#envRows tr').length===12);
 // Layout viewport sizes emulate 80%, 100%, 125% and 150% zoom, with both sidebar states.
 for(const [baseW,baseH] of [[1734,960],[1440,900],[1200,760],[1184,740]])for(const zoom of [.8,1,1.25,1.5])for(const collapsed of [false,true]){
  const width=Math.round(baseW/zoom),height=Math.round(baseH/zoom);await page.setViewportSize({width,height});
  await page.evaluate(value=>document.querySelector('.app-shell').classList.toggle('sidebar-collapsed',value),collapsed);await page.waitForTimeout(320);
  const boxes=await page.locator('#envRows tr').first().evaluate(row=>{const rect=e=>{const r=e.getBoundingClientRect();return {x:r.x,y:r.y,w:r.width,h:r.height,right:r.right,bottom:r.bottom}};const image=row.querySelector('.window-kernel-icon img');return {cell:rect(row.lastElementChild),logo:rect(image),launch:rect(row.querySelector('.launch')),gear:rect(row.querySelector('.dots')),loaded:image.naturalWidth>0,overflow:document.documentElement.scrollWidth>innerWidth}});
  assert(boxes.loaded,'logo asset missing');assert.equal(boxes.overflow,false,'page horizontal overflow '+JSON.stringify({baseW,zoom,collapsed,boxes}));
  assert(Math.abs(boxes.logo.w-boxes.logo.h)<.1,'logo distorted');assert(boxes.logo.x>=boxes.cell.x-1,'logo escaped action cell '+JSON.stringify({baseW,zoom,collapsed,boxes}));
  assert(boxes.logo.x>=boxes.launch.x&&boxes.logo.right<boxes.launch.right&&boxes.launch.right<boxes.gear.x,'actions overlap '+JSON.stringify({baseW,zoom,collapsed,boxes}));
  assert(boxes.gear.right<=width&&boxes.gear.right<=boxes.cell.right+1,'gear clipped '+JSON.stringify({baseW,zoom,collapsed,width,boxes}));assert(Math.abs(boxes.logo.y+boxes.logo.h/2-(boxes.launch.y+boxes.launch.h/2))<1,'vertical alignment');
  try{await gear().click({timeout:10000})}catch(error){throw Error('gear inaccessible '+JSON.stringify({baseW,baseH,zoom,collapsed,width,height,boxes,table:await page.locator('#environments>.table-wrap').boundingBox()})+' '+error.message)}const menu=await page.locator('#rowMenu').boundingBox();assert(menu.x>=0&&menu.y>=0&&menu.x+menu.width<=width+1&&menu.y+menu.height<=height+1,'menu clipped');await page.locator('#pageTitle').click();
 }
 assert.deepEqual(errors,[]);console.log('Engine icon, restart success/failure, settings edit/clone/delete, 64 viewport/zoom/sidebar layouts and menu boundaries passed');
}finally{await browser.close()}
