# One Browser 1.8.4 Windows x64 测试状态

构建日期：2026-09-15

## 已完成

- Windows 系统代理优先；未启用系统代理时自动回退直连。
- fingerprint-chromium 下载与 Clash 订阅导入优先使用系统代理。
- 当前 Wails 壳层接管关闭事件，避免右上角关闭按钮被拦截后失效。
- Windows WebView GPU 已启用；最大化和连续调整窗口尺寸时暂停毛玻璃动画与昂贵重绘。
- 代理节点只保留一个委托式点击监听，并加入单次删除锁。
- 删除当前代理节点后，界面自动恢复“系统代理优先”。
- 活跃界面和桌面桥接仅保留 fingerprint-chromium Windows x64。
- 工作台起始页展示窗口配置并实时检测出口 IP，第二标签打开 Google。
- 移除 Ant 指纹检测书签，并加入 `--no-default-browser-check`。
- 三个内置扩展会按启用状态下载安装，并设为新窗口默认扩展。
- Chrome 应用商店和 Google Drive 在运行中的受管浏览器标签页打开。
- 操作日志记录前端关键操作与后端错误，支持导出、30天清理及当前列表清空。
- 便携包不包含浏览器内核、账号、代理节点、日志、缓存或用户数据库。
- 便携包附带 Windows x64 Mihomo 代理运行时，用于启动用户导入的 Clash 节点。
- 代理删除显示为 X；筛选按钮取消外描边。
- Windows 主窗口、EXE 与托盘统一为 One Browser O 图标资源。
- 工作台出口 IP 与网络位置合并为同一信息组。
- Chromium 扩展开发者模式默认开启；持久扩展注册明确写入启用状态。
- 账号列表不再提供密码预览；侧栏账号区和更多按钮间距已调整。
- 帮助中心显示 1.8.3 版本号，删除右侧搜索控件和嵌套滚动区。
- One Browser 主界面和工作台均优先使用苹方字体。
- 每个窗口配置使用独立用户数据目录、调试端口并强制 `--new-window`，防止第二个配置复用第一个浏览器窗口。
- 工作台已改为紧凑指纹信息表，显示完整 User Agent、语言、时区、地理位置策略、分辨率、字体指纹和 WebRTC 状态。

## 已通过的检查

- 前端 TypeScript 编译与 Vite production build。
- JavaScript 语法检查。
- 代理删除模拟点击：连续两次点击只触发一次删除调用。
- 全量 `go test ./...` Go 单元测试。
- 全量 `go vet ./...` 静态检查。
- Wails Windows/amd64 production 交叉构建。
- PE 文件检查：主程序与 Mihomo 均为 PE32+ x86-64。

## 已知限制

- 当前执行环境为 Linux，没有 Windows 或 Wine，因此不能在此环境真实点击 Windows 原生关闭/最大化按钮，也不能读取真实 WinINet 注册表或启动 WebView2。
因此本包是已成功构建、通过全量自动化测试的 Windows x64 便携候选版，但不是 Windows 实机验收版。首次在 Windows 11 运行时，建议重点验证同时启动两个窗口时是否各自打开独立浏览器，以及关闭、最大化、系统代理和 fingerprint-chromium 下载。
