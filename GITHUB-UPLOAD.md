# One Browser 源码上传与构建

当前仓库代码版本为 2.0.7。实际下载版本以 GitHub Releases 为准。

## 上传代码

上传源码时保留 .gitignore，不提交账号、密码、数据库、Cookie、浏览器用户目录、代理订阅、内核二进制、node_modules 或构建产物。data/ 中只保留说明文件。

实际界面位于 frontend/public/one-browser/，React 桌面桥接入口为 frontend/src/OneBrowserShell.tsx。

## Windows x64 构建

准备 Go、Node.js / npm、Wails v2、Windows 构建工具及 Microsoft Edge WebView2 Runtime。

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
go mod download
npm --prefix frontend ci
npm --prefix frontend run test:one-browser
go test ./backend/... ./internal/...
wails build -platform windows/amd64 -clean -o OneBrowser.exe
python tools/verify-windows-resources.py build/bin/OneBrowser.exe
```

最后一项检查最终 EXE 的 Windows 启动资源，不能用未展开的清单模板直接替代。发布前还需在 Windows 上验证启动、缩放、托盘及浏览器操作。

## 发布文件

客户端只发布 Windows x64 完整版与源码：

- OneBrowser-2.0.7-windows-x64-portable.zip
- OneBrowser-2.0.7-source.zip

正式版本标签使用 v2.0.7 形式。未来版本替换为对应版本号，客户端升级检测依赖正式发布与正确的完整版资源名。

便携包保留必要运行组件和说明，不包含真实用户数据及浏览器内核。CAMOUFOX-TESTING.md、STARTUP-REVIEW.md、TESTING-STATUS.md 仅保留在源码中。

注意：旧 bat/publish.ps1 中仍有 AntBrowser 名称及跨平台流程，使用旧脚本前需核对其产物命名和打包内容，不应将旧脚本输出直接当作符合当前规范的发布包。

## 六张截图

上传到 docs/screenshots/。命名、排版模板和注意事项见 docs/screenshots/README.md。实际图片到齐后，再启用首页的两列三行图片区。

