# 1.8.23 UI 回退说明

1. 本次新增修正集中在 `frontend/public/one-browser/stage19-dark-surface-layout.css`。
2. 若只需恢复 1.8.22 的界面，删除 `index.html` 中该文件的引用即可；窗口、账号、代理和内核数据不会受影响。
3. “透明磨砂质感”仍可在系统设置中即时关闭，关闭后恢复基础视觉。
4. 如需继续回退更早的磨砂层，请按 `UI-ROLLBACK-1.8.22.md` 的顺序移除对应样式引用。
