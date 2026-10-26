(function(root,factory){const api=factory();if(typeof module==='object'&&module.exports)module.exports=api;else root.OneBrowserWindowIdentity=api})(typeof globalThis!=='undefined'?globalThis:this,function(){
  const validID=value=>typeof value==='string'&&/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
  function newID(){
    if(globalThis.crypto?.randomUUID)return globalThis.crypto.randomUUID();
    const bytes=new Uint8Array(16);globalThis.crypto.getRandomValues(bytes);bytes[6]=(bytes[6]&15)|64;bytes[8]=(bytes[8]&63)|128;
    const hex=Array.from(bytes,value=>value.toString(16).padStart(2,'0')).join('');return `${hex.slice(0,8)}-${hex.slice(8,12)}-${hex.slice(12,16)}-${hex.slice(16,20)}-${hex.slice(20)}`;
  }
  function normalize(entries,makeID=newID){
    const ids=new Set(),profiles=new Set();
    // New windows are prepended by the UI. Retain the original older row's
    // legacy profile and detach any newer rows accidentally bound to it.
    for(let i=entries.length-1;i>=0;i--){
      const entry=entries[i];let id=entry.windowId;
      if(!validID(id)||ids.has(id)){id=makeID();entry.windowId=id}
      ids.add(id);
      const profile=typeof entry.profileId==='string'?entry.profileId.trim():'';
      if(profile&&profiles.has(profile)){entry.profileId='';entry.state='stopped';entry.lastOpened=''}
      else if(profile){entry.profileId=profile;profiles.add(profile)}
    }
    return entries;
  }
  function clone(entry,makeID=newID){
    return {...entry,starting:false,restarting:false,pluginKeys:[],customPluginIDs:[],createdAt:new Date().toISOString(),windowId:makeID(),profileId:'',name:`${entry.name} 副本`,state:'stopped',lastOpened:'',used:'尚未启动',config:{...(entry.config||{})}};
  }
  function monthlyCreated(entries,now=new Date()){
    let count=0,unknown=0;
    for(const entry of entries){
      const raw=entry.createdAt,date=typeof raw==='string'&&/^\d{4}-\d{2}-\d{2}T/.test(raw)?new Date(raw):new Date(NaN);
      if(!Number.isFinite(date.getTime())){unknown++;continue}
      if(date<=now&&date.getFullYear()===now.getFullYear()&&date.getMonth()===now.getMonth())count++;
    }
    return {count,unknown};
  }
  return {newID,validID,normalize,clone,monthlyCreated};
});
