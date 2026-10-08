# 1.8.25 UI 回退说明

1. 本次界面修正集中在 `frontend/public/one-browser/stage21-theme-proxy-polish.css`。
2. 删除 `index.html` 中该文件的引用，即可恢复 1.8.24 的界面视觉。
3. 代理连接栈修复属于功能修复，不依赖此 CSS；新安装默认使用随包 Xray，已有可用 Mihomo 配置不会被修改。
4. 系统设置中的“透明磨砂质感”仍可即时关闭全部磨砂视觉，不影响业务数据。
