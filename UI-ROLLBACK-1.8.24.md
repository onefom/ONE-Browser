# 1.8.24 UI 回退说明

1. 本次界面修正集中在 `frontend/public/one-browser/stage20-responsive-glass-popovers.css`。
2. 删除 `index.html` 中该文件的引用，即可恢复 1.8.23 的主界面视觉。
3. 工作台的大屏缩放位于 `backend/onebrowser_workspace_page.go`；如需完整恢复 1.8.23，请使用源码包中上一个版本的同名文件。
4. 系统设置中的“透明磨砂质感”开关仍可即时关闭全部磨砂视觉，不影响业务数据。
