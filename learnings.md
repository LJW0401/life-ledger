# life-ledger Learnings

- 2026-07-04 阶段 0 完成，无 learnings（已执行反思清单）。

### 2026-07-04

#### [类型：架构洞察]
- **发现于**：WI-1.12 执行过程中
- **描述**：Playwright 测试依赖本机浏览器缓存，首次运行必须安装 Chromium；缺失时 E2E 会失败但服务和代码本身可正常工作。
- **建议处理方式**：在 README 开发准备步骤中固定 `npm --prefix web exec playwright install chromium`，CI 后续也需要显式安装浏览器。
- **紧急程度**：低
