# One Browser 2.0.5 · Camoufox 测试说明

## 开始测试

1. 从 https://github.com/daijro/camoufox/releases 下载 `win.x86_64.zip`，不要选 `win.i686`。
2. 在内核管理最上方导入内核包，或选择已解压且包含 `camoufox.exe` 的完整目录。
3. 建议新建一个测试窗口，在内核版本中选择 Camoufox。已有窗口也可停止后通过齿轮 → 编辑窗口切换。
4. 测试网页打开、表单提交、代理出口、账号登录，以及正常关闭后再打开、齿轮重启。
5. 如出现问题，请反馈内核版本、操作步骤、页面提示和系统日志中的 Camoufox 错误。

## 数据保存与隔离

- 窗口沿用自己的永久标识和 `OneBrowser-窗口名-短ID` 文件夹，Firefox 数据保存在其中的 `camoufox` 子目录。
- Chromium 原有数据保留在原处，两个引擎的数据格式不同，不自动转换或共享登录状态。
- 已启用持久 Cookie、密码保存提示、会话恢复。是否保持登录还取决于网站的会话有效期和浏览器内是否允许保存密码。
- 运行中阻止删除和切换内核；停止采用 Gecko 正常退出，超时提示手动关闭，不自动强杀。

## 当前支持范围

- Windows x64 Camoufox 归档/目录导入、版本显示、逐窗口选择和内核删除关联保护。
- 独立进程和私有 Juggler 管道，无需安装 Python/Node，也不开放 TCP 调试端口。
- 继续沿用当前代理连接栈；将结果映射为 Firefox HTTP/SOCKS5 设置，SOCKS5 使用远程 DNS，关闭 WebRTC。
- 支持选择 Windows、macOS、Linux 基础系统标识：Firefox UA、请求头、navigator.platform、oscpu 与 appVersion 统一匹配目标系统；语言、时区和每窗口音频种子保持独立。未引入完整 BrowserForge 设备指纹生成器，GPU、Canvas、媒体硬件等不保证完整模拟另一台设备。macOS UA 使用 Firefox 的固定兼容版本标识，不代表宿主系统已变为 macOS。
- 当前插件中心仍面向 Chrome 扩展。关联此类插件的 Camoufox 窗口会提示先取消关联；Firefox 扩展可先在 Camoufox 内安装。
- 内核在线检查更新仍针对 Fingerprint Chromium。Camoufox 当前采用手动导入更新，旧版有关联窗口时不能删除。
- 关闭 One Browser 主界面到托盘会继续保留窗口。完全退出管理程序时，Camoufox 私有控制管道断开，浏览器会随之关闭；“仅退出主程序并保留浏览器”暂仅适用于 Chromium。
- Chromium 的 CDP Cookie/自动化等接口尚未转换为 Firefox 接口。本次先测试手动浏览。

## 本次验证与限制

- 子进程集成测试验证私有管道、关闭/重启、稳定配置、运行中禁止删除、保留原 Chromium 和其他窗口的数据。
- 界面检查覆盖不同宽度、缩放对应视口和侧栏状态；按钮和齿轮保持可点击。
- 构建环境不是 Windows，未完成真实 Windows Camoufox 网站、插件和登录恢复实测，需用户反馈。

## 本次测试反馈与新增修正

用户已验证 Camoufox 打开、停止、重启、历史网页恢复、数据保存和多窗口隔离正常；同场景 Chromium 150 崩溃、148 正常。本次不自动切换或降级已有窗口的内核。

- Chrome / Firefox 内核分别分组，管理弹窗只包含同类版本。Camoufox 仍通过手动导入更新。
- 修改系统配置后，通过齿轮重启，新的系统参数及工作台信息才会生效；已打开网页可以在重启后刷新确认。
- Camoufox 启用按 profile 分组的任务栏设置；管理器也按永久 profile 标识设置独立任务栏标识。Windows 上实际显示仍需用户验证。
