One Browser 1.0 - Windows 64 位绿色版
====================================

使用方法
1. 将整个 ZIP 解压到本地文件夹，请勿直接在压缩包内运行。
2. 双击 One Browser.exe。
3. 首次启动默认没有账号、窗口和浏览器内核。
4. 账号、窗口、代理和外观设置保存在程序同级 data 文件夹。
5. 移动到另一台电脑时，请连同 data 文件夹一起复制。

主要功能
- 窗口管理：每个窗口使用独立的 Cookie、缓存和本地存储。
- 账号管理：账号名称、密码、备注和关联窗口统一管理。
- 内核管理：界面提供 Chromium、Microsoft Edge、Firefox 管理入口。
- 代理管理：支持 Clash、HTTP / SOCKS5 与链式代理配置。
- 插件中心：默认配置 AdGuard、篡改猴、Google 翻译；Firefox 不加载 Chromium 插件。
- 数据备份：本地备份保存在 data/backups。
- 启动窗口：显示“工作台配置”和“Google”两个标签。

数据目录
- data/one-browser-data.json：主要设置与账号、窗口数据。
- data/windows：独立窗口数据目录。
- data/backups：本地备份。
- data/extensions：插件启用记录。
- data/kernels：后续下载的浏览器内核目录。

注意事项
- 请保留整个程序目录，不要只复制 exe。
- 删除 data 文件夹会清除本地资料，删除前请先备份。
- Google Drive 正式同步需要配置 Google OAuth 客户端授权。
- 外部订阅、扩展和浏览器内核的下载受本机网络环境影响。
