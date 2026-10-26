# One Browser

One Browser 是面向 Windows x64 的本地多账号隔离浏览器管理工具，用于独立管理浏览器窗口、账号、代理、扩展程序和本地数据。

当前版本：`2.0.5`

2.0.5 移除插件卡片未关联时的重复提示；运行中窗口的配置图标间显示主色渐变双细线波动，停止后恢复静态显示。保留按窗口关联插件、会话恢复、代理和独立数据保护。

## 主要功能

- 每个浏览器窗口使用独立的 Cookie、缓存和本地存储。
- 仅使用 `fingerprint-chromium` Windows x64 浏览器内核。
- 浏览器内核由用户首次使用时按需下载，不随源码和便携包提供。
- Windows 系统代理优先，可自动使用 Clash 或 FlClash 当前系统代理。
- 支持导入 Clash 订阅、选择代理节点和删除节点。
- 删除当前代理节点后自动恢复 Windows 系统代理。
- 首次启动打开工作台和 Google；已有会话时恢复上次标签页。
- AdGuard、篡改猴和Google翻译默认不加载，点击后按窗口单独或批量关联。
- 提供账号管理、本地数据备份、系统日志和主题设置。
- Windows WebView GPU 已启用，并针对最大化和窗口缩放进行了性能优化。
- 账号、窗口、界面设置、数据快照、日志和 WebView2 本地状态统一保存在程序旁的 `data` 目录。

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

发布前校验 EXE 内的 Windows 启动资源：

```powershell
python tools/verify-windows-resources.py build/bin/OneBrowser.exe
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

当前后端回归、前端构建及交互模拟检查通过；Windows 实机验收范围见 `TESTING-STATUS.md`。

## 数据与隐私

仓库不应提交以下内容：

- 账号和密码
- Clash 订阅地址及代理节点
- 日志和数据库
- Cookie、缓存和浏览器用户目录
- fingerprint-chromium 浏览器内核
- `node_modules`、EXE 和发布压缩包

运行产生的数据统一保存在程序旁的 `data` 目录。空白便携包不会自动导入 Windows `%APPDATA%` 中的旧窗口；升级时请复制旧版的整个 `data` 文件夹。系统日志页仅显示当前进程会话，磁盘日志位于 `data/logs/app.log`。上传公开仓库前，请确认 `.gitignore` 未被删除。

## 项目结构

```text
backend/        Go 后端与桌面功能
frontend/       当前 One Browser UI、React 桌面桥接及生成绑定
build/          应用图标及 Windows 构建资源
scripts/        开发与测试脚本
test/           测试资料
wails.json      Wails 项目配置
config.yaml     默认配置
```

GitHub 上传和 Windows 构建的详细步骤请查看 `GITHUB-UPLOAD.md`。

## 当前版本文件

旧版发布说明、旧 React 页面及相关依赖已清理，只保留 `GITHUB-RELEASE-2.0.5.md`。Windows 启动清单已修正，不能将未展开的 Wails 模板直接嵌入 EXE。

## 2.0.0

当前版本 2.0.5，版本末位按十进制进位：2.0.9 → 2.1.0，2.9.9 → 3.0.0。执行 `python3 tools/bump-version.py` 可同步递增界面与构建元数据。功能变化见当前发布说明，验证范围见 TESTING-STATUS.md。

2.0.1：完整数字头像、精简账号卡片、固定表头层级、双向账号关联；进一步减少启动重复读写，显示实际用时并记录阶段。详见当前发布说明。

2.0.2：登录账号下拉滚动、深色停止标签、账号对应底色、品牌 Logo 上传和圆形裁切。详见当前发布说明。

2.0.3：本月新增日期统计、品牌Logo长条布局、启动链路分段计时与可选项目检查。仅提供完整版与源码，详见 STARTUP-REVIEW.md。

2.0.5：插件按窗口关联、可选重启、目录同步前台优先、旧兼容扫描按迁移执行。详见当前发布说明及STARTUP-REVIEW.md。

2.0.5 同版插件中心修订：顶部导入，三个内置插件优先，新增插件按顺序排列；统一浅色模板及窗口选择/停用，弹窗保留完整操作区。

导入插件直接点击名称编辑，Enter或点击别处保存，Escape放弃；自定义名称与首字母同步，重复导入保留名称。
