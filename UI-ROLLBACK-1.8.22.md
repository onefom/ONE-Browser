# 1.8.22 透明磨砂 UI 回退说明

1. 在 One Browser 中打开“系统设置”。
2. 关闭“透明磨砂质感”，界面会立即恢复 1.8.17 的视觉样式。
3. 如需只恢复 1.8.21 的控件视觉，可移除 `stage18-control-consistency.css` 的引用。
4. 如需继续恢复 1.8.20 的表单和菜单视觉，可移除 `stage17-input-motion-polish.css` 的引用。
5. 如需恢复 1.8.19 的磨砂效果，再移除 `stage16-glass-refinement.css` 的引用。
6. 如需从源码永久移除全部磨砂视觉，删除 `stage15-restrained-glass.css`、`stage16-glass-refinement.css`、`stage17-input-motion-polish.css` 和 `stage18-control-consistency.css` 的引用；功能修复不会受到影响。

四个增量样式文件没有覆盖或删除 1.8.17 的原始 UI 文件，可按以上层级逐步回退。
