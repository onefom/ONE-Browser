const {app,BrowserWindow,ipcMain,shell}=require('electron');
const path=require('path');
const fs=require('fs');

const appRoot=path.resolve(__dirname,'..');
const portableRoot=app.isPackaged?path.dirname(app.getPath('exe')):appRoot;
const dataRoot=path.join(portableRoot,'data');
const windowsRoot=path.join(dataRoot,'windows');

function ensureDataFolders(){
  [dataRoot,windowsRoot,path.join(dataRoot,'backups'),path.join(dataRoot,'extensions'),path.join(dataRoot,'kernels')].forEach(dir=>fs.mkdirSync(dir,{recursive:true}));
}

function writeJson(file,value){
  ensureDataFolders();
  const target=path.join(dataRoot,file);
  const temp=target+'.tmp';
  fs.writeFileSync(temp,JSON.stringify(value,null,2),'utf8');
  fs.renameSync(temp,target);
  return target;
}

function createMainWindow(){
  const win=new BrowserWindow({
    width:1440,height:900,minWidth:1024,minHeight:680,show:false,
    backgroundColor:'#eef0f7',title:'One Browser',autoHideMenuBar:true,
    webPreferences:{preload:path.join(__dirname,'preload.js'),contextIsolation:true,nodeIntegration:false,sandbox:true}
  });
  win.loadFile(path.join(appRoot,'dist','index.html'));
  win.once('ready-to-show',()=>win.show());
  win.webContents.setWindowOpenHandler(({url})=>{shell.openExternal(url);return{action:'deny'}});
}

app.whenReady().then(()=>{ensureDataFolders();createMainWindow();app.on('activate',()=>{if(BrowserWindow.getAllWindows().length===0)createMainWindow()})});
app.on('window-all-closed',()=>{if(process.platform!=='darwin')app.quit()});

ipcMain.handle('data:save',(_event,snapshot)=>writeJson('one-browser-data.json',snapshot));
ipcMain.handle('extensions:save',(_event,payload)=>writeJson('extensions/enabled.json',payload));
ipcMain.handle('backup:save',(_event,bytes,name)=>{
  ensureDataFolders();
  const safe=String(name||'OneBrowser-data.zip').replace(/[^a-zA-Z0-9._-]/g,'_');
  const target=path.join(dataRoot,'backups',safe);
  fs.writeFileSync(target,Buffer.from(bytes));
  return target;
});
ipcMain.handle('browser:open',(_event,profile)=>{
  const id=String(profile?.id??Date.now()).replace(/[^a-zA-Z0-9_-]/g,'_');
  const win=new BrowserWindow({width:1280,height:820,minWidth:900,minHeight:620,title:profile?.name||'One Browser',autoHideMenuBar:true,backgroundColor:'#f5f6fa',webPreferences:{partition:`persist:one-browser-${id}`,contextIsolation:true,nodeIntegration:false,sandbox:false,webviewTag:true}});
  const config=encodeURIComponent(JSON.stringify(profile||{}));
  win.loadFile(path.join(__dirname,'workspace.html'),{query:{config}});
  return true;
});
ipcMain.handle('system:openExternal',(_event,url)=>shell.openExternal(url));
