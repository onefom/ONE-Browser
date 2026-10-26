import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import vm from 'node:vm';
const source=await readFile(new URL('../public/one-browser/app.js',import.meta.url),'utf8');
const functionLine=name=>source.split('\n').find(line=>line.startsWith(`function ${name}(`)||line.startsWith(`async function ${name}(`));
const button={disabled:false,textContent:'',dataset:{}};
const nodes=[{id:'fast',name:'fast'},{id:'slow',name:'slow'}];
let callback,releaseBatch,unsubscribed=false;
let batch=new Promise(resolve=>{releaseBatch=resolve});
const context={
 document:{querySelector:()=>button},
 managedKernelKey:'chrome',kernelState:{chrome:true},kernelVersions:{chrome:'150.0.7871.186'},latestKernelRelease:{version:'150.0.7871.186'},
 familyRelease:()=>context.latestKernelRelease,familyCurrent:()=>context.kernelVersions.chrome,
 importedClashNodes:nodes,proxyLatencyRefreshing:false,
 desktopBridge:()=>({onProxySpeedResult:fn=>{callback=fn;return ()=>{unsubscribed=true}},testProxyNodes:()=>batch}),
 renderProxyNodes:()=>{},showToast:()=>{},
};
vm.createContext(context);
for(const name of ['numericVersion','compareKernelVersions','updateKernelManagerAction','applyProxyLatencyResult','refreshProxyLatency']) {
 const line=functionLine(name);assert(line,`missing ${name}`);vm.runInContext(line,context);
}
vm.runInContext('updateKernelManagerAction()',context);
assert.equal(button.disabled,false);assert.equal(button.textContent,'检查更新');
context.kernelVersions.chrome='151.0.0.1';vm.runInContext('updateKernelManagerAction()',context);assert.equal(button.disabled,false);
context.kernelVersions.chrome='149.0.0.1';vm.runInContext('updateKernelManagerAction()',context);assert.equal(button.disabled,false);
button.dataset.updating='true';button.textContent='正在更新';vm.runInContext('updateKernelManagerAction()',context);assert.equal(button.textContent,'正在更新');delete button.dataset.updating;
let pending=vm.runInContext('refreshProxyLatency()',context);
assert.deepEqual(nodes.map(n=>n.latencyState),['testing','testing']);
callback({proxyId:'fast',ok:true,latencyMs:35});
assert.deepEqual(nodes.map(n=>n.latencyState),['ok','testing'],'fast node must show before the batch finishes');
releaseBatch([{proxyId:'fast',ok:true,latencyMs:35},{proxyId:'slow',ok:false,error:'timeout'}]);await pending;
assert.deepEqual(nodes.map(n=>n.latencyState),['ok','timeout']);assert.equal(unsubscribed,true);assert.equal(button.disabled,false);assert.equal(context.proxyLatencyRefreshing,false);
// A batch error must not erase a result already delivered by the stream.
unsubscribed=false;let rejectBatch;batch=new Promise((_,reject)=>{rejectBatch=reject});pending=vm.runInContext('refreshProxyLatency()',context);
callback({proxyId:'fast',ok:true,latencyMs:20});rejectBatch(new Error('batch disconnected'));await pending;
assert.equal(nodes[0].latencyMs,20);assert.equal(nodes[0].latencyState,'ok');assert.equal(nodes[1].latencyState,'timeout');assert.equal(unsubscribed,true);
console.log('One Browser 1.9 update guard and streaming latency checks passed');
