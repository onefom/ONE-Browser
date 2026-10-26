import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';
import ts from 'typescript';
const identityContext={module:{exports:{}}};
vm.runInNewContext(await readFile(new URL('../public/one-browser/window-identity.js',import.meta.url),'utf8'),identityContext);
const identity=identityContext.module.exports;
let next=1;const makeID=()=>`11111111-1111-4111-8111-${String(next++).padStart(12,'0')}`;
const old={name:'original',profileId:'old-profile',state:'running'};
const entries=[old];identity.normalize(entries,makeID);const oldID=old.windowId;
const fresh={name:'new',windowId:makeID(),profileId:'',state:'stopped'};entries.unshift(fresh);identity.normalize(entries,makeID);
assert.equal(old.windowId,oldID);assert.equal(fresh.profileId,'');
entries.reverse();entries.reverse();assert.equal(old.windowId,oldID);
const copy=identity.clone(old,makeID);assert.notEqual(copy.windowId,oldID);assert.equal(copy.profileId,'');assert.equal(copy.state,'stopped');
const mistaken={name:'newer wrongly bound',profileId:'old-profile',state:'running'};
identity.normalize([mistaken,old],makeID);assert.equal(mistaken.profileId,'');assert.equal(mistaken.state,'stopped');assert.equal(old.profileId,'old-profile');
const idsBefore=JSON.stringify(entries.map(e=>e.windowId));identity.normalize(entries,makeID);assert.equal(JSON.stringify(entries.map(e=>e.windowId)),idsBefore,'IDs must survive reload');
const storage=new Map([['one-browser-ant-profile-map',JSON.stringify({'0':'old-profile'})]]);
const calls=[],events=new Map();let created=0;
const context={exports:{},Promise,Map,Set,JSON,String,Error,console,
 localStorage:{getItem:key=>storage.get(key)||null,setItem:(key,value)=>storage.set(key,value)},
 document:{title:''},
 require:name=>name==='react'?{useEffect:fn=>fn()}:name==='react/jsx-runtime'?{jsx:()=>null}:null,
 window:{go:{main:{App:{
  OneBrowserStart:async input=>{calls.push(['start',input]);await new Promise(resolve=>setTimeout(resolve,2));return {profileId:input.profileId||`created-${++created}`,running:true}},
  OneBrowserStop:async id=>calls.push(['stop',id]),
  OneBrowserGetNetworkInfo:async id=>calls.push(['network',id]),
 }}},runtime:{EventsOn:(name,fn)=>{events.set(name,fn);return ()=>events.delete(name)}}},
};
const shellSource=await readFile(new URL('../src/OneBrowserShell.tsx',import.meta.url),'utf8');
const compiled=ts.transpileModule(shellSource,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2020}}).outputText;
vm.runInNewContext(compiled,context);context.exports.default();const bridge=context.window.onlineDesktop;
bridge.registerBrowserWindows(entries);
const result=await bridge.startBrowserWindow({id:fresh.windowId,profileId:'',name:fresh.name,browserKey:'chrome'});
assert.equal(calls[0][1].windowId,fresh.windowId);assert.equal(calls[0][1].profileId,'','new row must not inherit numeric row-zero mapping');
fresh.profileId=result.profileId;bridge.registerBrowserWindows(entries);
await bridge.stopBrowserWindow(oldID);assert.deepEqual(calls.at(-1),['stop','old-profile']);
await bridge.getWindowNetworkInfo(fresh.windowId);assert.deepEqual(calls.at(-1),['network',fresh.profileId]);
let closedID;bridge.onBrowserClosed(id=>closedID=id);events.get('browser:instance:stopped')('old-profile');assert.equal(closedID,oldID,'close event must use permanent identity');
await bridge.startBrowserWindow({id:oldID,profileId:old.profileId,name:'renamed',browserKey:'chrome'});
assert.equal(calls.at(-1)[1].profileId,'old-profile','renaming must retain only the same window profile');
const a=makeID(),b=makeID();await Promise.all([bridge.startBrowserWindow({id:a,browserKey:'chrome'}),bridge.startBrowserWindow({id:b,browserKey:'chrome'})]);
const persisted=JSON.parse(storage.get('one-browser-window-profiles-v2'));assert(persisted[a]);assert(persisted[b]);assert.notEqual(persisted[a],persisted[b]);
await assert.rejects(()=>bridge.startBrowserWindow({id:0,browserKey:'chrome'}));
const css=await readFile(new URL('../public/one-browser/one-browser-current.css',import.meta.url),'utf8');
const gridRule=css.slice(css.lastIndexOf('#proxy #proxyNodeGrid{'));assert(gridRule.includes('background:transparent!important'));assert(gridRule.includes('backdrop-filter:none!important'));
console.log('One Browser independent identity, migration, cloning, concurrent mapping and proxy background checks passed');
