# One Browser 仓库清理建议

核对对象：onefom/ONE-Browser 默认分支，2026-10-26。此文件是清理建议，不表示远程文件已经删除。

## 应当保留

- 主程序及构建入口：backend/、frontend/、internal/、build/、main.go、go.mod、go.sum、wails.json、config.yaml、OneBrowser.ico。
- Git 规则：.gitignore、.gitattributes。
- 当前说明：README.md、CHANGELOG.md、GITHUB-RELEASE-2.0.7.md、README-FIRST.txt。
- AGENTS.md：开发协作约定，建议保留。
- tools/：包含版本更新、资源校验及运行组件工具，不能整体删除。

## 可以精简的旧发布内容

本项目当前交付目标是 Windows x64。若不再维护 Linux/macOS 发布，可移除：

- .github/workflows/publish-linux.yml
- .github/workflows/publish-macos.yml

已读取这两份文件：Linux 工作流会在 v* 标签推送时触发，并包含 amd64/arm64 打包和 Release 上传；macOS 工作流提供手动打包。

publish/linux/、publish/mac/、对应平台默认配置，以及 runtime 清单中的其他平台条目可以进一步精简，但必须连同发布脚本的对应入口和引用一起修改，不能只删目录。

## 适合归档的开发资料

建议将 CAMOUFOX-TESTING.md、STARTUP-REVIEW.md、TESTING-STATUS.md 移入 docs/，同步更新 README、发布脚本及其他引用。它们保留在源码中，不放入客户端便携包。

GITHUB-UPLOAD.md 内容仍写 1.9，需要更新。本上传包提供了面向当前版本的替换稿。

## 需要确认用途的候选项

- skills/：旧 Ant Browser / OpenClaw 接入资料与安装脚本；确认不再使用或维护该接入后，可归档或删除。
- test/automa/：Automa 示例与导入测试资料；可按是否保留示例决定，删除前检查测试和文档引用。
- bat/、scripts/：包含有效开发、构建、恢复和辅助入口，逐个整理，不能因为文件名带旧名称就整目录删除。

## publish 不能整目录删除

已读取 bat/publish.ps1，Windows 打包仍引用：

- publish/config.init.yaml
- publish/runtime-manifest.json
- publish/installer.nsi（安装包分支）

脚本还引用 publish/linux/publish-linux-docker.ps1。精简跨平台发布时需同步修改这个入口。当前发布脚本仍存在 AntBrowser 包名，需要另行与 OneBrowser 的发布命名规则统一。

## data 目录

当前 GitHub 的 data/ 只有 README.md，可以保留。用户电脑上运行产生的 data/ 包含浏览器与账号数据，不属于本次仓库清理范围。

## 执行状态

GitHub 连接读取成功，但创建分支返回 403（Resource not accessible by integration）。尚未创建清理分支、提交修改或删除远程文件。本包可以手动上传；写入权限恢复后再通过分支统一处理引用和验证。

