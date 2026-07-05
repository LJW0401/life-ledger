<!--
release.template.md - 发布说明模板；发版前复制为 release.md 并填写。
release-version 必须与将要推送的 git tag 完全一致，Release CI 会校验。
-->
<!-- release-version: <VERSION> -->

# 发布 <VERSION>

> **对比基线**：<上一个发布 tag，例如 `v0.0.1-preview`> · 全部改动见 [`<PREV_VERSION>...<VERSION>`](https://github.com/<OWNER>/<REPO>/compare/<PREV_VERSION>...<VERSION>)
> <首个版本无基线可比时，整段删掉。>

<可选：一两句话总览，说明本次发布的主题；没什么可写就整段删掉，不要留空。>

## 新功能

- <从用户视角写新增能力。必要时点出对应页面、API 或命令名。>
- <一行一事，多个改动拆成多条。>

## 修复

- <描述修复了什么用户可感知的问题，避免只写 commit 哈希。优先点出复现路径或之前会触发什么错误现象。>

## 其他改进

- <内部重构、文档、工具链里值得用户知道的部分；纯仓库清理写不动就整段删掉。>

## 破坏性变更

- <如有：用三段写清楚影响什么、为什么改、迁移办法。>
- <涉及 `config.toml` schema、SQLite 数据迁移、Excel 导入导出列格式、登录设备/session 语义任一不兼容时必须写在这里。>
- <无破坏性变更时整段（含标题）删掉，不要留“无”占位。>

## 附件

`.github/workflows/release.yml` 在 tag 触发时会构建并上传 Linux 单文件二进制：

- `life-ledger-linux-amd64`：Linux x86_64
- `life-ledger-linux-arm64`：Linux arm64

本地可用 `make build` 产出当前平台单文件二进制到 `bin/life-ledger`。发布前应确认 Release 附件包含当前版本的全部 Linux 二进制。

## 升级指南

- **覆盖二进制**：用新 `life-ledger` 可执行文件替换旧文件即可。
- **配置文件**：`config.toml` 应继续放在二进制同目录；除非本版本在“破坏性变更”中说明配置 schema 调整，否则无需手动迁移。
- **数据目录**：相对路径的 `data.dir` 和 `backup.dir` 按 `config.toml` 所在目录解析，升级时不要移动现有 `data/` 和 `backups/`。
- **备份**：升级前建议运行 `./life-ledger backup`，确认备份包包含 SQLite 数据库、`config.toml` 和 `backup-meta.json`。
- **启动校验**：替换后启动服务，确认登录页可访问，账单、重要日期和决策页面能正常打开。
