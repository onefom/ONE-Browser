import assert from 'node:assert/strict';
const {chromium}=await import(process.env.PLAYWRIGHT_MODULE||'playwright');
const browser=await chromium.launch({executablePath:process.env.CHROMIUM_EXECUTABLE,headless:true,args:['--no-sandbox','--disable-dev-shm-usage','--allow-file-access-from-files']});
try{
 for(const [width,height] of [[1734,961],[900,680]]){
  const page=await browser.newPage({viewport:{width,height}}),errors=[];
  page.on('pageerror',error=>errors.push(error.message));
  await page.addInitScript(()=>{
   localStorage.clear();localStorage.setItem('one-browser-accounts',JSON.stringify(Array.from({length:30},(_,i)=>({service:String(i+1),identity:'test-password-'+(i+1)}))));
   window.onlineDesktop={getKernelStatus:async()=>({chrome:{installed:false}}),registerBrowserWindows:()=>{}};
  });
  await page.goto(new URL('../public/one-browser/index.html',import.meta.url).href);
  await page.locator('#currentAccount').click();await page.locator('#loginAccountModal.open').waitFor();
  const trigger=page.locator('#loginAccountSelect').locator('..').locator('.rounded-select-trigger'),list=page.locator('.rounded-select-popover');
  await trigger.click();
  const bounds=await list.boundingBox();assert(bounds.y>=0&&bounds.y+bounds.height<=height,'options must fit viewport');
  assert.equal(await list.locator('.rounded-select-option').count(),30,'all accounts must exist');
  await list.hover();await page.mouse.wheel(0,4000);await page.waitForFunction(()=>{const e=document.querySelector('.rounded-select-popover');return e.classList.contains('open')&&e.scrollTop>0&&Math.abs(e.scrollTop-(e.scrollHeight-e.clientHeight))<2});
  assert((await list.getAttribute('class')).includes('open'),'own scroll must not close the dropdown');
  const scroll=await list.evaluate(e=>({top:e.scrollTop,max:e.scrollHeight-e.clientHeight}));assert(scroll.top>0&&Math.abs(scroll.top-scroll.max)<2);
  await list.locator('[data-value="29"]').click();
  assert.equal(await page.locator('#loginAccountSelect').inputValue(),'29');assert.equal(await page.locator('#loginAccountPassword').inputValue(),'test-password-30');
  assert.equal(await trigger.getAttribute('aria-expanded'),'false');
  await trigger.click();await page.waitForTimeout(80);
  assert((await list.getAttribute('class')).includes('open'),'opening a later selection must remain open');
  const selectedBox=await list.locator('[data-value="29"]').boundingBox(),listBox=await list.boundingBox();
  assert(selectedBox.y>=listBox.y&&selectedBox.y+selectedBox.height<=listBox.y+listBox.height,'selected late account must be visible');
  await page.keyboard.press('Escape');
  assert.equal(await page.locator('#loginAccountModal.open').count(),1,'Esc closes only the dropdown first');
  const submit=page.locator('#loginAccountForm [type="submit"]'),button=await submit.boundingBox();assert(button.y+button.height<=height);
  await submit.click();assert.equal(await page.locator('#currentAccount strong').textContent(),'30');
  assert.deepEqual(errors,[]);await page.close();
 }
 console.log('Login account scroll passed: 30 accounts, wheel to last, selected item on reopen, correct credentials and login at large/minimum viewport sizes');
}finally{await browser.close()}
