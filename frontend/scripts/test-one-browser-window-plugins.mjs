import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import vm from 'node:vm';
import ts from 'typescript';
const modelContext={};vm.createContext(modelContext);vm.runInContext(await readFile(new URL('../public/one-browser/window-plugins.js',import.meta.url),'utf8'),modelContext);
const model=modelContext.OneBrowserWindowPlugins,windows=[{name:'A'},{name:'B',pluginKeys:['adguard','bad','adguard'],customPluginIDs:['bad','aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa']}];
model.normalize(windows);assert.equal(windows[0].pluginKeys.length,0);assert.equal(windows[0].customPluginIDs.length,0);assert.equal(windows[1].pluginKeys.join(','),'adguard');assert.equal(windows[1].customPluginIDs.length,1);
const identityContext={module:{exports:{}}};vm.runInNewContext(await readFile(new URL('../public/one-browser/window-identity.js',import.meta.url),'utf8'),identityContext);
const clone=identityContext.module.exports.clone(windows[1],()=> 'copy');assert.equal(clone.pluginKeys.length,0);assert.equal(clone.customPluginIDs.length,0);assert.equal(windows[1].pluginKeys.length,1);
const calls=[],storage=new Map();let finishStart;
const startGate=new Promise(resolve=>finishStart=resolve);
const context={exports:{},Promise,Map,Set,JSON,String,Error,console,setTimeout,document:{title:''},
 localStorage:{getItem:key=>storage.get(key)||null,setItem:(key,value)=>storage.set(key,value)},
 require:name=>name==='react'?{useEffect:fn=>fn()}:name==='react/jsx-runtime'?{jsx:()=>null}:null,
 window:{go:{main:{App:{
  OneBrowserStart:async input=>{calls.push(['start',input]);await startGate;return {profileId:'active-profile',running:true}},
  OneBrowserRenameWindow:async(...args)=>{calls.push(['directory',...args])},
  OneBrowserSetWindowPlugins:async(...args)=>{calls.push(['plugins',...args]);return{}},
 }}},runtime:{EventsOn:()=>()=>{}}},
};
const compiled=ts.transpileModule(await readFile(new URL('../src/OneBrowserShell.tsx',import.meta.url),'utf8'),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2020}}).outputText;
vm.runInNewContext(compiled,context);context.exports.default();const bridge=context.window.onlineDesktop;
const start=bridge.startBrowserWindow({id:'window-a',browserKey:'chrome',extensions:['adguard'],extensionIds:['aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa']});
const background=bridge.syncWindowDirectories([{windowId:'window-b',profileId:'profile-b',name:'B'},{windowId:'window-c',profileId:'profile-c',name:'C'}]);
await new Promise(resolve=>setTimeout(resolve,130));assert.equal(calls.filter(c=>c[0]==='directory').length,0,'background directories must yield while a foreground start is pending');
finishStart();await start;await background;assert.equal(calls.filter(c=>c[0]==='directory').length,2);
assert.equal(calls[0][1].extensions.join(','),'adguard');assert.equal(calls[0][1].extensionIds.join(','),'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa');
await bridge.setWindowPlugins({id:'window-b',profileId:'profile-b',keys:['google-translate'],extensionIds:[]});assert.equal(calls.at(-1)[1],'window-b');assert.equal(calls.at(-1)[2],'profile-b');assert.equal(calls.at(-1)[3].join(','),'google-translate');
context.window.go.main.App.OneBrowserStart=async()=>{throw new Error('simulated failed start')};
await assert.rejects(()=>bridge.startBrowserWindow({id:'failed-window',browserKey:'chrome'}),/simulated failed start/);
await bridge.syncWindowDirectories([{windowId:'after-error',profileId:'profile-after-error',name:'After failure'}]);
assert.equal(calls.at(-1)[1],'after-error','failed foreground start must release background queue');
console.log('Window plugin defaults, strict keys, clone opt-out, per-window bridge payload and foreground directory priority passed');
