# One Browser

One Browser 是面向 Windows x64 的本地多账号隔离浏览器管理工具，用于独立管理浏览器窗口、账号、代理、扩展程序和本地数据。

当前版本：`1.8.4`

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

运行产生的数据默认保存在本地 `data` 目录。上传公开仓库前，请确认 `.gitignore` 未被删除。

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
