import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--disable-dev-shm-usage','--allow-file-access-from-files']});
try{
 const page=await browser.newPage({viewport:{width:1734,height:961}});
 const errors=[];page.on('pageerror',error=>errors.push(error.message));
 await page.addInitScript(()=>{
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:false}}),syncWindowDirectories:async()=>[],registerBrowserWindows:()=>{},logOperation:async()=>{}};
  if(localStorage.getItem('__fixture'))return;
  localStorage.clear();localStorage.setItem('__fixture','1');
  localStorage.setItem('one-browser-environments',JSON.stringify(Array.from({length:20},(_,i)=>({windowId:crypto.randomUUID(),profileId:'',name:'窗口 '+(i+1),note:'测试',owner:'未关联账号',state:'stopped',config:{browser:'chrome'},tag:'新环境'}))));
  localStorage.setItem('one-browser-accounts',JSON.stringify(['1','2','3','4','5','6','7','8','9','10','333'].map(service=>({service,identity:'saved-secret',tone:'dark'}))));
  window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:false}}),syncWindowDirectories:async()=>[],registerBrowserWindows:()=>{},logOperation:async()=>{}};
 });
 await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);await page.waitForTimeout(120);
 await page.locator('[data-page="accounts"]').click();
 assert.equal(await page.locator('.account-password').count(),0,'password placeholder must be removed');
 assert.equal(await page.locator('#accountGrid .service-icon').nth(9).textContent(),'10');
 assert.equal(await page.locator('#accountGrid .service-icon').nth(10).textContent(),'333');
 assert((await page.locator('#accountGrid .account-card').first().boundingBox()).height<=84,'card must be compact');
 await page.locator('[data-page="environments"]').click();
 const add=page.locator('#envRows tr').first().locator('[data-action="profile"]');
 await add.click();
 const accountID=await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-accounts')).find(account=>account.service==='2').accountId);
 await page.locator(`#profileMenu [data-account-id="${accountID}"]`).click();
 await add.click();
 const selected=page.locator(`#profileMenu [data-account-id="${accountID}"]`);
 assert.equal(await selected.getAttribute('aria-pressed'),'true');assert((await selected.textContent()).includes('已关联'));
 await page.locator('[data-page="accounts"]').click();
 assert((await page.locator('#accountGrid .account-card').nth(1).textContent()).includes('窗口 1'));
 await page.locator('[data-page="environments"]').click();
 await page.locator('#envRows tr').first().locator('[data-action="rename"]').click();
 await page.locator('.inline-name-input').fill('改名后');await page.locator('.inline-name-input').press('Enter');
 await page.locator('[data-page="accounts"]').click();
 assert((await page.locator('#accountGrid .account-card').nth(1).textContent()).includes('改名后'));
 await page.reload();await page.locator('[data-page="environments"]').click();await add.click();
 assert.equal(await selected.getAttribute('aria-pressed'),'true','selection must survive reload');
 await page.locator('#profileMenu [data-profile-action="unlink"]').click();
 await page.locator('[data-page="accounts"]').click();
 assert((await page.locator('#accountGrid .account-card').nth(1).textContent()).includes('未关联窗口'));
 assert.equal(await page.evaluate(()=>JSON.parse(localStorage.getItem('one-browser-accounts'))[1].identity),'saved-secret','hiding placeholders must preserve credentials');
 await page.locator('[data-page="environments"]').click();await page.waitForTimeout(350);
 const list=page.locator('#environments>.table-wrap');
 for(const offset of [10,30,50,72,100,150,350,800]){
  await list.evaluate((e,value)=>e.scrollTop=value,offset);await page.waitForTimeout(30);
  const header=page.locator('#environments th').nth(3),box=await header.boundingBox();
  const onHeader=await page.evaluate(({x,y})=>Boolean(document.elementFromPoint(x,y)?.closest('th')),{x:box.x+box.width/2,y:box.y+box.height/2});
  assert(onHeader,`row controls covered sticky header at scroll ${offset}`);
 }
 assert.deepEqual(errors,[]);
 console.log('Chromium UI: full numeric badges, compact cards, hidden password row, synced account selection, reload/rename/unlink and sticky-header stacking passed');
}finally{await browser.close()}
