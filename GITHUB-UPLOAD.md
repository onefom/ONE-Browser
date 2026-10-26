# One Browser 1.9 源码上传与构建

源码只保留当前 One Browser 界面。`frontend/src/main.tsx` 加载桌面桥接 `OneBrowserShell.tsx`，实际界面位于 `frontend/public/one-browser/`；`wailsjs` 为生成的桌面接口绑定。旧 React 页面、路由、主题和相关依赖已移除。

## 上传 GitHub

在源码目录初始化仓库并提交，或通过 GitHub Desktop 添加源码目录并发布。不要提交 `data` 个人数据、浏览器内核、运行时二进制或构建产物。`.gitignore` 保留这些排除规则。

## Windows x64 构建

安装 Go、Node.js 和 Wails v2，执行：

```powershell
cd frontend
npm ci
npm run test:one-browser
npm run build:clean
cd ..
go test ./backend/... ./internal/...
wails build -platform windows/amd64 -clean -o OneBrowser.exe
python tools/verify-windows-resources.py build/bin/OneBrowser.exe
```

最后一项检查最终 EXE 内的 Windows 启动资源。必须通过后才能打包。Windows manifest 使用明确的程序集名称及四段数字版本，不能把尚未展开的模板直接嵌入 EXE。

版本显示为 1.9；构建版本为 1.9.0，Windows 程序集版本为 1.9.0.0。
