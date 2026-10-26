(function(root){
 'use strict';
 function normalize(accounts,windows,makeID,migrateLegacy=false){
  const ids=new Set();
  for(const account of accounts){if(typeof account.accountId!=='string'||!account.accountId||ids.has(account.accountId))account.accountId=makeID();ids.add(account.accountId)}
  const byID=new Map(accounts.map(account=>[account.accountId,account]));
  if(migrateLegacy){
   for(const entry of windows){
    if(Object.prototype.hasOwnProperty.call(entry,'accountId'))continue;
    const match=accounts.filter(account=>account.service===entry.owner);
    if(match.length===1)entry.accountId=match[0].accountId;
   }
   for(const entry of windows){
    if(Object.prototype.hasOwnProperty.call(entry,'accountId'))continue;
    const match=accounts.filter(account=>account.window===entry.name&&windows.filter(item=>item.name===entry.name).length===1);
    if(match.length===1)entry.accountId=match[0].accountId;
   }
  }
  for(const entry of windows){
   const account=byID.get(entry.accountId);
   if(account)entry.owner=account.service;
   else if(entry.accountId){entry.accountId='';entry.owner='未关联账号'}
  }
  for(const account of accounts){
   const linked=windows.filter(entry=>entry.accountId===account.accountId);
   account.windowIds=linked.map(entry=>entry.windowId);
   account.window=linked.map(entry=>entry.name).join('、');
  }
  return accounts;
 }
 function accountIdForName(accounts,name,currentID=''){
  const current=accounts.find(account=>account.accountId===currentID&&account.service===name);
  if(current)return current.accountId;
  const matches=accounts.filter(account=>account.service===name);
  return matches.length===1?matches[0].accountId:'';
 }
 function assign(entry,account){entry.accountId=account?.accountId||'';entry.owner=account?.service||'未关联账号'}
 root.OneBrowserAccountLinks={normalize,assign,accountIdForName};
})(typeof window!=='undefined'?window:globalThis);
