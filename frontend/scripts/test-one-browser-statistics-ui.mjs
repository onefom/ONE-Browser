import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--disable-dev-shm-usage','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1440,height:900}}),errors=[];
 page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(()=>{
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:true,version:'150'}}),registerBrowserWindows:()=>{},deleteBrowserWindow:async()=>{}};
  if(localStorage.getItem('__statsFixture'))return;
  localStorage.clear();localStorage.setItem('__statsFixture','1');
  const now=new Date(),previous=new Date(now.getFullYear(),now.getMonth(),0,12);
  localStorage.setItem('one-browser-environments',JSON.stringify([{name:'Current',createdAt:now.toISOString()},{name:'Previous',createdAt:previous.toISOString()},{name:'Unknown'}].map(entry=>({...entry,windowId:crypto.randomUUID(),profileId:'',state:'stopped',config:{browser:'chrome'},tag:'新环境'}))));
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);await page.waitForTimeout(150);
 const stats=page.locator('#metricMonthCreated');assert.equal(await stats.textContent(),'本月 +1 · 1个日期未知');
 await page.locator('#createBtn').click();await page.locator('#envName').fill('New Window');await page.locator('#createForm [type="submit"]').click();
 assert.equal(await stats.textContent(),'本月 +2 · 1个日期未知');
 const created=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0].createdAt);assert(Number.isFinite(Date.parse(created)));
 await page.locator('#envRows tr').first().locator('[data-action="menu"]').click();await page.locator('#rowMenu [data-menu-action="clone"]').click();
 assert.equal(await stats.textContent(),'本月 +3 · 1个日期未知');
 await page.locator('#envRows tr').first().locator('[data-action="rename"]').click();await page.locator('.inline-name-input').fill('Renamed Window');await page.locator('.inline-name-input').press('Enter');
 assert.equal(await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-environments'))[0].createdAt),created,'rename must preserve original date');
 await page.locator('#search').fill('nothing-matches');assert.equal(await stats.textContent(),'本月 +3 · 1个日期未知');await page.locator('#search').fill('');
 await page.locator('#envRows tr').first().locator('[data-action="menu"]').click();await page.locator('#rowMenu [data-menu-action="delete"]').click();await page.locator('#confirmDeleteWindow').click();
 await page.waitForFunction(()=>document.querySelector('#metricMonthCreated').textContent.startsWith('本月 +2'));
 await page.reload();assert.equal(await stats.textContent(),'本月 +2 · 1个日期未知');
 await page.locator('[data-page="settings"]').click();
 for(const width of [1440,900]){
  await page.setViewportSize({width,height:900});await page.waitForTimeout(160);
  const row=await page.locator('.brand-logo-picker').boundingBox(),input=await page.locator('#productName').boundingBox(),upload=await page.locator('#chooseBrandLogo').boundingBox(),reset=await page.locator('#resetBrandLogo').boundingBox();
  assert(Math.abs(row.width-input.width)<3,'logo control must fill the same field width');
  assert(upload.x<reset.x&&Math.abs(upload.y+upload.height/2-reset.y-reset.height/2)<2,'actions must share one horizontal row');
  assert(reset.x+reset.width<=row.x+row.width&&reset.x+reset.width>=row.x+row.width-12,'restore at the right edge');
 }
 assert.deepEqual(errors,[]);
 console.log('Statistics/brand UI passed: create, clone, rename, filter, delete, reload, unknown legacy dates and full-width logo row at two widths');
}finally{await browser.close()}
