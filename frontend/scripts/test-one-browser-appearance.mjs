import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import vm from 'node:vm';
const context={};vm.runInNewContext(await readFile(new URL('../public/one-browser/account-appearance.js',import.meta.url),'utf8'),context);
const normalize=context.OneBrowserAccountAppearance.normalize;
const accounts=Array.from({length:12},(_,index)=>({service:String(index+1),tone:['blue','pink','dark'][index%3]}));
normalize(accounts);assert.equal(new Set(accounts.slice(0,9).map(item=>item.colorHue)).size,9);
const original=accounts.map(item=>item.colorHue);normalize(accounts);assert.deepEqual(accounts.map(item=>item.colorHue),original);
accounts.splice(1,1);const remaining=accounts.map(item=>item.colorHue);accounts.push({service:'new'});normalize(accounts);
assert.deepEqual(accounts.slice(0,-1).map(item=>item.colorHue),remaining);
assert.equal(new Set(accounts.map(item=>item.colorHue)).size,accounts.length);
accounts.push({service:'invalid',colorHue:'0);background:black'},{service:'duplicate',colorHue:accounts[0].colorHue});normalize(accounts);
assert(accounts.every(item=>Number.isFinite(item.colorHue)&&item.colorHue>=0&&item.colorHue<360));
assert.equal(new Set(accounts.map(item=>item.colorHue)).size,accounts.length);
console.log('Account colors: nine distinct shades, stable migration and deletion, safe values passed');

assert.equal(context.OneBrowserAccountAppearance.badgeLabel('10'),'10');assert.equal(context.OneBrowserAccountAppearance.badgeLabel('333'),'333');assert.equal(context.OneBrowserAccountAppearance.badgeLabel('管理员'),'管');
