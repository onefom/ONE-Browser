import assert from 'node:assert/strict';
import {mkdir} from 'node:fs/promises';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--disable-dev-shm-usage','--allow-file-access-from-files']});
try{
 for(const [width,height] of [[1734,961],[1440,900],[1200,760],[900,680]]){
  const page=await browser.newPage({viewport:{width,height}});
  const errors=[];page.on('pageerror',error=>errors.push(error.message));
  await page.addInitScript(()=>{
   localStorage.clear();
   localStorage.setItem('one-browser-environments',JSON.stringify(Array.from({length:30},(_,i)=>({windowId:crypto.randomUUID(),profileId:'',name:'窗口 '+(i+1),note:'测试备注',state:'stopped',config:{browser:'chrome'},tag:'新环境',initial:String(i+1)}))));
   localStorage.setItem('one-browser-accounts',JSON.stringify(Array.from({length:12},(_,i)=>({service:String(i+1),identity:'test',tone:['blue','pink','dark'][i%3]}))));
   window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:false}}),syncWindowDirectories:async()=>[],registerBrowserWindows:()=>{},logOperation:async()=>{}};
  });
  await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);
  await page.waitForTimeout(150);
  const list=page.locator('#environments>.table-wrap'),heading=page.locator('main>header'),metrics=page.locator('.metrics');
  const beforeHeader=await heading.boundingBox(),beforeMetrics=await metrics.boundingBox();
  const size=await list.evaluate(e=>({height:e.clientHeight,scrollHeight:e.scrollHeight,bar:getComputedStyle(e).scrollbarWidth}));
  assert(size.height>90&&size.scrollHeight>size.height,`${width}: rows must overflow within a bounded list`);
  assert.equal(size.bar,'none');
  await list.hover();await page.mouse.wheel(0,100000);await page.waitForTimeout(200);
  assert.equal((await heading.boundingBox()).y,beforeHeader.y,'header moved');
  assert.equal((await metrics.boundingBox()).y,beforeMetrics.y,'metrics moved');
  const bounds=await list.boundingBox(),last=await page.locator('#envRows tr').last().boundingBox();
  assert(last&&last.y>=bounds.y&&last.y+last.height<=bounds.y+bounds.height+1,'last row must be fully reachable');
  const head=await page.locator('#environments th').first().boundingBox();
  assert(Math.abs(head.y-bounds.y)<=2,'table header must stick');
  await page.locator('[data-page="accounts"]').click();await page.waitForTimeout(100);
  const colors=await page.locator('#accountGrid .service-icon').evaluateAll(nodes=>nodes.slice(0,9).map(e=>getComputedStyle(e).backgroundImage));
  assert.equal(new Set(colors).size,9,'first nine accounts need distinct colors');
  assert.deepEqual(errors,[]);
  if(process.env.UI_QA_DIR){await mkdir(process.env.UI_QA_DIR,{recursive:true});await page.screenshot({path:`${process.env.UI_QA_DIR}/accounts-${width}.png`});}
  await page.close();
 }
 console.log('Chromium layout checks passed: 30 rows, sticky header, hidden scrollbars and nine account colors at four viewport sizes');
}finally{await browser.close()}
