# One Browser

One Browser 是面向 Windows x64 的本地多账号隔离浏览器管理工具，用于独立管理浏览器窗口、账号、代理、扩展程序和本地数据。

当前版本：`1.8.12`

## 主要功能

- 每个浏览器窗口使用独立的 Cookie、缓存和本地存储。
- 仅使用 `fingerprint-chromium` Windows x64 浏览器内核。
- 浏览器内核由用户首次使用时按需下载，不随源码和便携包提供。
- Windows 系统代理优先，可自动使用 Clash 或 FlClash 当前系统代理。
- 支持导入 Clash 订阅、选择代理节点和删除节点。
- 删除当前代理节点后自动恢复 Windows 系统代理。
- 启动窗口时打开工作台配置页和 Google 首页。
- 默认支持 AdGuard、篡改猴和 Google 翻译扩展。
- 提供账号管理、本地数据备份、系统日志和主题设置。
- Windows WebView GPU 已启用，并针对最大化和窗口缩放进行了性能优化。
- 账号、窗口、界面设置、数据快照、日志和 WebView2 本地状态统一保存在程序旁的 `data` 目录。

## # Changelog

## [1.8.13] - 2026-09-17

* 首页网络详情与浏览器实际代理链路同步，显示真实出口 IP 和位置。
* 代理节点增加延迟检测，显示延迟或 `Timeout`，支持手动及每 60 秒自动刷新。
* 移除“使用此节点”按钮，点击节点卡片即可选中。
* 首页筛选支持按内核版本、分组和代理情况组合筛选。
* 优化帮助中心滚动、代理弹窗滚动条及节点主题配色。

## [1.8.12] - 2026-09-16

* 将内核下载按钮移动到页面右上角，内核卡片显示当前内核版本。
* 应用版本号移动到系统设置右上角，改为纯文字显示。
* 窗口图标、内核图标、按钮和渐变效果跟随主题颜色。
* 自定义颜色改为应用内圆角配色面板，修复弹出位置问题。
* 首页网络详情精简为“出口 IP”和“位置”。

## [1.8.11] - 2026-09-16

* 修复 AdGuard、Tampermonkey 和 Google 翻译未正确加载的问题。
* 增加插件下载备用地址、完整性校验和失败重试。
* 首次启动窗口时等待插件准备完成，并默认开启扩展开发者模式。
* 修复侧边栏折叠时底部账号区域跳动和文字竖排问题。
* 统一各页面标题、间距及右上角操作按钮的位置和高度。

## [1.8.10] - 2026-09-16

* 调整菜单顺序为：窗口、内核、代理、账号、插件、日志、设置。
* 内核升级和云备份入口仅保留在窗口管理首页。
* 修复窗口列表中的系统、网络、账号及更多操作按钮。
* 统一各页面间距、按钮高度和表格布局，减少切换时的视觉跳动。
* 完善系统日志导出、清理及 `data/logs/app.log` 保存位置说明。


## 技术栈

- Go
- Wails v2
- React
- TypeScript
- Vite
- SQLite

## 开发环境

建议使用 Windows 11 x64，并安装：

- Go
- Node.js 与 npm
- Wails CLI v2
- Microsoft Edge WebView2 Runtime

安装 Wails CLI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 安装依赖

```powershell
cd frontend
npm install
cd ..
go mod download
```

## 开发运行

```powershell
wails dev
```

## 构建 Windows x64

```powershell
wails build -platform windows/amd64 -clean -o OneBrowser.exe
```

生成的程序位于：

```text
build/bin/OneBrowser.exe
```

## 测试

```powershell
cd frontend
npm run build
npm run test:one-browser
cd ..
go test ./backend/internal/browser ./backend
go vet ./backend/internal/browser ./backend
```

源码基线中有两个外部内核备份导入测试可能失败，详细情况请查看 `TESTING-STATUS.md`。

## 数据与隐私

仓库不应提交以下内容：

- 账号和密码
- Clash 订阅地址及代理节点
- 日志和数据库
- Cookie、缓存和浏览器用户目录
- fingerprint-chromium 浏览器内核
- `node_modules`、EXE 和发布压缩包

运行产生的数据统一保存在程序旁的 `data` 目录。1.8.12 空白便携包不会再自动导入 Windows `%APPDATA%` 中的旧窗口；升级时请复制旧版的整个 `data` 文件夹。系统日志页仅显示当前进程会话，磁盘日志位于 `data/logs/app.log`。上传公开仓库前，请确认 `.gitignore` 未被删除。

## 项目结构

```text
backend/        Go 后端与桌面功能
frontend/       React 前端与 One Browser UI
build/          应用图标及 Windows 构建资源
scripts/        开发与测试脚本
test/           测试资料
wails.json      Wails 项目配置
config.yaml     默认配置
```

GitHub 上传和 Windows 构建的详细步骤请查看 `GITHUB-UPLOAD.md`。
