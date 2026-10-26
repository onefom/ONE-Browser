import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1440,height:900}}),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.addInitScript(()=>{localStorage.clear();window.__ticks=[];const old=window.setInterval;window.setInterval=(fn,ms,...args)=>{if(ms===60000)window.__ticks.push(fn);return old(fn,ms,...args)};window.starts=0;window.pending=null;window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:false},camoufox:{installed:false}}),listKernels:async()=>[],registerBrowserWindows:()=>{},syncWindowDirectories:async()=>[],logOperation:async()=>{},startIPMonitor:async()=>{if(++window.starts===1)throw new Error('startup not ready')},getIPChange:async()=>window.pending,onIPChange:()=>{},resolveIPChange:async()=>{window.pending=null}}});
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);assert.equal(await page.locator('#ipChangeModal.open').count(),0);
 const sample={id:'native-pending',before:{ip:'192.0.2.18',country:'德国'},after:{ip:'198.51.100.26',country:'美国'},windows:[{windowId:'a',name:'窗口01'}]};
 await page.evaluate(async sample=>{window.pending=sample;await window.__ticks.at(-1)()},sample);await page.waitForSelector('#ipChangeModal.open');assert((await page.evaluate(()=>window.starts))>=2,'startup failure must retry');
 for(const dark of [false,true]){await page.evaluate(dark=>document.body.classList.toggle('dark-content',dark),dark);const styles=await page.locator('[data-ip-before],[data-ip-after]').evaluateAll(nodes=>nodes.map(el=>{const s=getComputedStyle(el);return{weight:Number(s.fontWeight),color:s.color}}));assert.equal(styles[0].weight,400);assert.equal(styles[1].weight,700);assert.notEqual(styles[0].color,styles[1].color)}
 await page.locator('[data-ip-continue]').click();assert.equal(await page.locator('#ipChangeModal.open').count(),0);assert.deepEqual(errors,[]);
 console.log('UI18: native startup retry, missed-event pending fallback at 60 seconds, one dialog, bold coloured IPs in light/dark, continue passed');
}finally{await browser.close()}
