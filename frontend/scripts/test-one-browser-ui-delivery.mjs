import assert from 'node:assert/strict';
import {createServer} from 'node:http';
import {readFileSync,existsSync,readdirSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import {resolve,extname} from 'node:path';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const root=fileURLToPath(new URL('../',import.meta.url));
const sourceCSS=readFileSync(resolve(root,'public/one-browser/one-browser-current.css'),'utf8');
const oldCSS=sourceCSS.split('/* One Browser 2.0.6: shared dark surfaces.')[0];
const directories=readdirSync(resolve(root,'dist')).filter(n=>/^one-browser-[a-f0-9]{12}$/.test(n));
assert.equal(directories.length,1);assert(!existsSync(resolve(root,'dist/one-browser')));
const packaged=readFileSync(resolve(root,'dist',directories[0],'index.html'),'utf8');
assert(packaged.includes('id="one-browser-dark-appearance"'));assert(packaged.includes('data-ui-revision="'+directories[0]+'"'));
const server=createServer((req,res)=>{
 const path=new URL(req.url,'http://localhost').pathname;
 const file=path.startsWith('/baseline/')?resolve(root,'public',path.slice(10)):resolve(root,'dist',path.slice(1)||'index.html');
 if(!file.startsWith(root)||!existsSync(file)){res.writeHead(404);res.end();return}
 res.setHeader('Content-Type',({'.html':'text/html; charset=utf-8','.css':'text/css; charset=utf-8','.js':'text/javascript; charset=utf-8','.svg':'image/svg+xml'})[extname(file)]||'application/octet-stream');
 res.end(readFileSync(file));
});
await new Promise(r=>server.listen(0,'127.0.0.1',r));const origin='http://127.0.0.1:'+server.address().port;
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox']});
const styles=async(frame,selectors)=>frame.evaluate(selectors=>selectors.map(s=>{
 const el=document.querySelector(s);if(!el)throw new Error('missing '+s);const c=getComputedStyle(el);
 return{selector:s,bg:c.backgroundColor,image:c.backgroundImage,color:c.color,border:c.borderTopColor,font:c.fontSize,weight:c.fontWeight};
}),selectors);
const lum=s=>{const c=s.match(/[\d.]+/g).map(Number);return .2126*c[0]+.7152*c[1]+.0722*c[2]};
try{
 const context=await browser.newContext({viewport:{width:1440,height:980}});
 await context.addInitScript(()=>{
  if(!localStorage.getItem('delivery-seeded')){
   localStorage.clear();localStorage.setItem('delivery-seeded','1');localStorage.setItem('one-browser-kernel-release-cache',JSON.stringify({checkedAt:Date.now(),release:{version:'151.0.0.1'}}));
   localStorage.setItem('one-browser-accounts',JSON.stringify([{accountId:'22222222-2222-4222-8222-222222222222',service:'账号01',identity:'test-only'}]));
  }
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:true,version:'150.0.7871.186'},camoufox:{installed:true,version:'158.0'}}),listKernels:async()=>[
   {coreId:'chrome144',version:'144.0.7559.132',family:'chromium',installed:true},
   {coreId:'firefox9',version:'158.0-beta.9',family:'camoufox',installed:true},
   {coreId:'chrome150',version:'150.0.7871.186',family:'chromium',installed:true},
   {coreId:'chrome148',version:'148.0.7778.215',family:'chromium',installed:true},
   {coreId:'firefox157',version:'157.0-beta.2',family:'camoufox',installed:true},
   {coreId:'chrome150old',version:'150.0.7871.9',family:'chromium',installed:true},
   {coreId:'firefox40',version:'158.0-beta.40',family:'camoufox',installed:true},
   {coreId:'firefoxstable',version:'158.0',family:'camoufox',installed:true}],getClientVersion:async()=>'2.0.7',checkClientUpdate:async()=>({version:'1.8.25',message:'本地版本较新'}),getLatestKernelRelease:async()=>({version:'151.0.0.1'}),registerBrowserWindows:()=>{},saveLocalData:async()=>{},logOperation:async()=>{}};
 });
 const page=await context.newPage(),errors=[];page.on('pageerror',e=>errors.push(e.message));
 await page.goto(origin+'/');await page.locator('iframe').waitFor();
 assert.match(await page.locator('iframe').getAttribute('src'),/^\/one-browser-[a-f0-9]{12}\/index.html$/);
 const frame=page.frames().find(f=>f.parentFrame());await frame.waitForSelector('#createBtn');
 const baseline=await context.newPage();await baseline.goto(origin+'/baseline/one-browser/index.html');
 const formSelectors=['#modal .modal','#modal .modal-head','#modal .advanced','#modal .quick-preset','#envName','#applyPreset','#configDetail','#modal .create-modal-actions'];
 // Compare the delivered document against the untouched light-mode source,
 // including glass on/off, dialogs and page cards, not just the colour tokens.
 for(const glass of [true,false]){
  for(const f of [frame,baseline])await f.evaluate(glass=>{document.body.classList.remove('dark-content');document.body.classList.toggle('restrained-glass',glass)},glass);
  const pageSelectors=['.metrics article','.settings-card','.account-card','.safe'];assert.deepEqual(await styles(frame,pageSelectors),await styles(baseline,pageSelectors));
  for(const f of [frame,baseline])await f.locator('#createBtn').click();assert.deepEqual(await styles(frame,formSelectors),await styles(baseline,formSelectors));
  for(const f of [frame,baseline])await f.locator('#cancelModal').click();
 }
 // Shuffled installation input must yield numeric newest-first display everywhere.
 const chromeVersions=['150.0.7871.186','150.0.7871.9','148.0.7778.215','144.0.7559.132'];
 const firefoxVersions=['158.0','158.0-beta.40','158.0-beta.9','157.0-beta.2'];
 const coreIDs=['chrome150','chrome150old','chrome148','chrome144','firefoxstable','firefox40','firefox9','firefox157'];
 for(const width of [1734,1184]){
  await page.setViewportSize({width,height:980});await frame.locator('[data-page="kernels"]').click();
  assert.deepEqual(await frame.locator('[data-family="chrome"] .installed-core-heading b').allTextContents(),chromeVersions);
  assert.deepEqual(await frame.locator('[data-family="camoufox"] .installed-core-heading b').allTextContents(),firefoxVersions);
  const cards=frame.locator('[data-family="chrome"] .installed-kernel-card'),a=await cards.nth(0).boundingBox(),b=await cards.nth(1).boundingBox();
  if(width>1300){assert(Math.abs(a.y-b.y)<2);assert(a.x<b.x)}else{assert(a.y<b.y);assert(Math.abs(a.x-b.x)<2)}
 }
 await frame.locator('.header-actions .kernel-download').click();await frame.locator('#kernelManagerModal.open').waitFor();
 assert.deepEqual(await frame.locator('#kernelVersionSelect option').evaluateAll(xs=>xs.map(x=>x.value)),['151.0.0.1',...chromeVersions]);
 assert.equal(await frame.locator('#kernelVersionSelect').inputValue(),'150.0.7871.186');await frame.locator('#closeKernelManager').click();
 await frame.locator('[data-page="environments"]').click();await frame.locator('#createBtn').click();
 assert.deepEqual(await frame.locator('#browserVersion option').evaluateAll(xs=>xs.map(x=>x.value)),coreIDs);
 await frame.locator('#envName').fill('排序绑定测试');
 await frame.locator('#browserVersion').locator('..').locator('.rounded-select-trigger').click();await frame.locator('.rounded-select-option[data-value="chrome144"]').click();
 await frame.locator('#submitEnv').click();await frame.locator('#modal').waitFor({state:'hidden'});
 const saved=await frame.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0]);assert.equal(saved.config.coreId,'chrome144');
 await page.reload();await page.locator('iframe').waitFor();const reloaded=page.frames().find(f=>f.parentFrame());await reloaded.waitForSelector('.row-actions .dots');
 await reloaded.locator('.row-actions .dots').click();await reloaded.locator('[data-menu-action="edit"]').click();
 assert.equal(await reloaded.locator('#browserVersion').inputValue(),'chrome144');assert.deepEqual(await reloaded.locator('#browserVersion option').evaluateAll(xs=>xs.map(x=>x.value)),coreIDs);
 await reloaded.locator('#cancelModal').click();assert.equal(await reloaded.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0].config.coreId),'chrome144');
 await reloaded.evaluate(()=>localStorage.removeItem('one-browser-environments'));
 // Force the old external sheet from before either dark fix. Critical styles
 // in the production HTML must still win after a full reload.
 await context.route('**/one-browser-current.css*',route=>route.fulfill({contentType:'text/css',body:oldCSS}));
 await page.reload();await page.locator('iframe').waitFor();const guarded=page.frames().find(f=>f.parentFrame());await guarded.waitForSelector('#createBtn');
 for(const width of [1734,1184,790])for(const glass of [true,false]){
  await page.setViewportSize({width,height:960});
  // Use the settings switches, rather than manually faking the resulting body class.
  await guarded.locator('[data-page="settings"]').click();
  await guarded.locator('#darkMode').evaluate(el=>{if(!el.checked)el.click()});
  await guarded.locator('#restrainedGlassMode').evaluate((el,glass)=>{if(el.checked!==glass)el.click()},glass);
  assert(await guarded.locator('body').evaluate(el=>el.classList.contains('dark-content')));
  await guarded.locator('[data-page="environments"]').click();
  // An empty placeholder must not paint the top 250px of a taller table.
  const emptyCell=guarded.locator('.window-empty-row td');
  for(const hover of [false,true]){
   if(hover)await emptyCell.hover();else await guarded.locator('#createBtn').hover();
   const empty=await emptyCell.evaluate(el=>{const s=getComputedStyle(el);return{bg:s.backgroundColor,image:s.backgroundImage,border:s.borderBottomWidth,shadow:s.boxShadow}});
   assert.equal(empty.bg,'rgba(0, 0, 0, 0)');assert.equal(empty.image,'none');assert.equal(empty.border,'0px');assert.equal(empty.shadow,'none');
  }
  await guarded.locator('#createBtn').click();
  const values=await styles(guarded,formSelectors);for(const v of values){assert(lum(v.bg)<95,JSON.stringify(v));assert(lum(v.color)>115,JSON.stringify(v));assert(!v.image.includes('255, 255, 255'),JSON.stringify(v))}
  assert.notEqual(values[0].bg,values[2].bg);assert.notEqual(values[2].bg,values[3].bg);
  for(const id of ['applyPreset','configDetail']){const button=guarded.locator('#'+id);await button.scrollIntoViewIfNeeded();await button.hover();const v=(await styles(guarded,['#'+id]))[0];assert(lum(v.bg)<95);assert(lum(v.color)>150);assert.equal(v.image,'none')}
  await guarded.locator('#configDetail').click();await guarded.locator('#configDetailModal.open').waitFor();await guarded.locator('#closeConfigDetail').click();await guarded.locator('#cancelModal').click();
  await guarded.locator('[data-page="accounts"]').click();const safe=(await styles(guarded,['.safe']))[0];assert(lum(safe.bg)<170);assert(lum(safe.color)>130);
  await guarded.locator('[data-page="environments"]').click();for(const id of ['cloudBackupBtn','coreUpdateBtn']){await guarded.locator('#'+id).click();const pop=id==='cloudBackupBtn'?'#cloudBackupPopover':'#coreUpdatePopover';const v=(await styles(guarded,[pop+' .up-to-date']))[0];assert.equal(v.bg,'rgba(0, 0, 0, 0)');await page.keyboard.press('Escape')}
  await guarded.locator('[data-page="kernels"]').click();await guarded.locator('.header-actions .kernel-download').click();const v=(await styles(guarded,['#deleteKernel','#kernelManagerModal .installed-pill']));for(const item of v)assert(lum(item.bg)<170,JSON.stringify(item));await guarded.locator('#closeKernelManager').click();
 }
 // Switching back to light must restore source styles even after the stale-sheet case.
 await guarded.locator('[data-page="settings"]').click();await guarded.locator('#darkMode').evaluate(el=>el.click());
 await guarded.locator('[data-page="environments"]').click();
  // An empty placeholder must not paint the top 250px of a taller table.
  const emptyCell=guarded.locator('.window-empty-row td');
  for(const hover of [false,true]){
   if(hover)await emptyCell.hover();else await guarded.locator('#createBtn').hover();
   const empty=await emptyCell.evaluate(el=>{const s=getComputedStyle(el);return{bg:s.backgroundColor,image:s.backgroundImage,border:s.borderBottomWidth,shadow:s.boxShadow}});
   assert.equal(empty.bg,'rgba(0, 0, 0, 0)');assert.equal(empty.image,'none');assert.equal(empty.border,'0px');assert.equal(empty.shadow,'none');
  }
  await guarded.locator('#createBtn').click();
 await baseline.evaluate(()=>{document.body.classList.remove('dark-content','restrained-glass')});await baseline.locator('#createBtn').click();assert.deepEqual(await styles(guarded,formSelectors),await styles(baseline,formSelectors));
 assert.deepEqual(errors,[]);console.log('Kernel version order: shuffled input, numeric patch comparison, Firefox stable/beta ordering, management dropdown, two/one-column cards, create/edit selection and persisted core binding passed');console.log('Production iframe: fingerprinted document/assets, embedded dark rules with deliberately stale external CSS, 6 dark layouts, live mode switches, real buttons/popovers, and light-mode before/after equality passed');
}finally{await browser.close();await new Promise(r=>server.close(r))}
