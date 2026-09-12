const {contextBridge,ipcRenderer}=require('electron');

contextBridge.exposeInMainWorld('onlineDesktop',{
  saveLocalData:(snapshot)=>ipcRenderer.invoke('data:save',snapshot),
  loadExtensions:(windowIndex,plugins)=>ipcRenderer.invoke('extensions:save',{windowIndex,plugins,updatedAt:new Date().toISOString()}),
  startBrowserWindow:(profile)=>ipcRenderer.invoke('browser:open',profile),
  saveLocalBackup:async(blob,name)=>ipcRenderer.invoke('backup:save',new Uint8Array(await blob.arrayBuffer()),name),
  connectGoogleDrive:()=>ipcRenderer.invoke('system:openExternal','https://accounts.google.com/'),
  uploadGoogleDriveBackup:async(blob,name)=>ipcRenderer.invoke('backup:save',new Uint8Array(await blob.arrayBuffer()),name)
});
