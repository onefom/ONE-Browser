# One Browser 1.8.3 源码上传与构建

## 上传 GitHub

1. 解压源码 ZIP，进入 `OneBrowser-1.8.3-source` 文件夹。
2. 在 GitHub 新建一个空仓库，不勾选自动创建 README。
3. 在源码目录打开 PowerShell，执行：

```powershell
git init
git add .
git commit -m "One Browser 1.8.3"
git branch -M main
git remote add origin https://github.com/你的用户名/你的仓库名.git
git push -u origin main
```

如果使用 GitHub Desktop，选择“Add an Existing Repository”，指向解压后的源码目录，再发布到 GitHub 即可。

## Windows x64 构建

需要安装 Go、Node.js、Wails v2，以及 Wails Windows 构建所需环境。然后在源码根目录执行：

```powershell
cd frontend
npm install
npm run build
cd ..
wails build -platform windows/amd64 -clean -o OneBrowser.exe
```

生成文件位于 `build/bin/OneBrowser.exe`。

## 源码包排除内容

本源码包不包含 `node_modules`、编译产物、便携发布包、账号、代理节点、日志、缓存、数据库、浏览器内核或用户工作区。依赖请通过 `npm install` 和 `go mod download` 获取，fingerprint-chromium 内核由软件首次使用时下载。
