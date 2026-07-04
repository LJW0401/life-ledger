# life-ledger Learnings

- 2026-07-04 阶段 0 完成，无 learnings（已执行反思清单）。

### 2026-07-04

#### [类型：架构洞察]
- **发现于**：WI-1.12 执行过程中
- **描述**：Playwright 测试依赖本机浏览器缓存，首次运行必须安装 Chromium；缺失时 E2E 会失败但服务和代码本身可正常工作。
- **建议处理方式**：在 README 开发准备步骤中固定 `npm --prefix web exec playwright install chromium`，CI 后续也需要显式安装浏览器。
- **紧急程度**：低

#### [类型：架构洞察]
- **发现于**：WI-2.10 执行过程中
- **描述**：阶段 2 后 `/api/*` 默认先经过认证，未登录请求会先返回 401；未知 API 的 404 只应在登录态有效后断言。
- **建议处理方式**：后续 API 测试按“认证边界优先、业务路由其次”的顺序组织用例。
- **紧急程度**：低

#### [类型：Bug]
- **发现于**：WI-3.5 执行过程中
- **描述**：Go nil slice 会编码为 JSON `null`，前端按数组读取时会在空列表场景崩溃。
- **建议处理方式**：API 列表响应初始化为空 slice，前端读取列表字段时也使用空数组兜底。
- **紧急程度**：中

#### [类型：Bug]
- **发现于**：WI-3.5 执行过程中
- **描述**：React 表单提交中异步 await 后继续使用 `event.currentTarget`，会导致后续 reset 或刷新逻辑不稳定。
- **建议处理方式**：提交函数开始时先保存 `const formElement = event.currentTarget`，后续只使用保存的 DOM 引用。
- **紧急程度**：低

#### [类型：架构洞察]
- **发现于**：WI-4.9 执行过程中
- **描述**：账单页保存预算后涉及写请求、统计刷新和预算列表刷新，E2E 如果只等待点击动作会出现断言早于刷新完成的竞态。
- **建议处理方式**：涉及写后刷新的 E2E 用例先等待对应 API 响应，再断言页面状态。
- **紧急程度**：低

#### [类型：Bug]
- **发现于**：WI-5.5 执行过程中
- **描述**：API 测试直接构造 `config.Config` 时不会经过 `config.Load` 的默认值填充，导致 Excel 上传大小限制为 0 并错误返回 413。
- **建议处理方式**：测试夹具直接构造配置时必须显式填写会影响行为的默认值；后续可考虑提供统一测试配置 helper。
- **紧急程度**：低

#### [类型：Bug]
- **发现于**：WI-6.5 执行过程中
- **描述**：页面存在多个同名操作按钮时，E2E 使用全页 `getByRole` 容易点到错误区域；写请求后立即断言也会和列表刷新产生竞态。
- **建议处理方式**：E2E 操作按钮先限定在对应 `aria-label` 区域内，再等待目标 API 响应状态，最后断言用户可见结果。
- **紧急程度**：低

#### [类型：架构洞察]
- **发现于**：WI-7.4 执行过程中
- **描述**：账单列表在分页结果内逐条查询标签会形成 N+1 查询，race 门控会把这类性能问题放大并暴露出来。
- **建议处理方式**：列表页加载跨表关联数据时默认使用批量查询；统计接口优先使用 SQL 聚合，不复用带展示字段装配的列表路径。
- **紧急程度**：中

#### [类型：Bug]
- **发现于**：快速功能：修复审核问题
- **描述**：构建路径、配置默认路径和测试文档在实现演进后发生漂移，用户按旧说明运行时会遇到配置缺失或空配置错误。
- **建议处理方式**：改变发布/启动约定时同步 PRD、SAD、TPD、部署文档和 release-report；为首次本地运行提供 `init-config` 这类可执行初始化入口。
- **紧急程度**：中

### Bug 修复：二次审核数据完整性问题
- **发现于**：二次代码审核文档 `review-notes/code-review-2026-07-04-rerun.md`
- **现象**：账单导出遗漏分页外数据、备份直接复制 SQLite 文件、Excel 金额格式错误缺少行级详情、交易 JSON 必填布尔字段可省略、决策复盘日期在本地当天 08:00 前不生效。
- **根因**：测试只覆盖了正常路径和小数据量，没有覆盖超过分页上限、运行中数据库快照、字段 presence、金额语法边界和 date-only 本地日界线。
- **修复**：导出改用无分页查询；备份改用 SQLite `VACUUM INTO` 快照；Excel 解析复用交易金额规则；API 请求层用指针布尔识别字段缺失；决策状态按本地日期比较。
- **回归测试**：`internal/api/api_test.go`、`internal/backup/backup_test.go`、`internal/excel/excel_test.go`、`internal/domain/decisions/decisions_test.go`。
- **为什么原测试没覆盖**：原测试验证了功能可用，但没有把契约里的“全量”“一致性”“行级错误”“必填字段”和“日期语义”变成边界断言。
- **紧急程度**：高

### Bug 修复：决策复盘时区来源不稳定
- **发现于**：新版本代码审核文档 `review-notes/code-review-2026-07-04-new-version.md`
- **现象**：决策复盘日期按 date-only 比较后仍使用主机 `time.Local`，云服务器系统时区为 UTC 时，Asia/Shanghai 当天 00:00 到 07:59 仍不会进入 `待复盘`。
- **根因**：领域逻辑读取进程本地时区，没有把 `config.toml` 中的应用时区作为显式依赖注入。
- **修复**：配置加载阶段校验 `export.timezone`，API 组装层加载该时区并注入 `decisions.Service`，决策状态计算只使用注入时区。
- **回归测试**：`internal/domain/decisions/decisions_test.go`、`internal/config/config_test.go`。
- **为什么原测试没覆盖**：测试通过手动设置 `time.Local` 复现本地日期边界，遗漏了“主机时区和应用配置时区不同”的部署场景。
- **紧急程度**：中

### 快速功能：GitHub CI 和 Release 自动化
- **类型**：架构洞察
- **描述**：GitHub Actions 的 tag release 工作流必须先存在于默认分支，之后推送新标签才会触发；release 还需要 `release.md` 中的 `release-version` 和 pushed tag 完全一致。
- **建议处理方式**：先合并工作流到 `main`，再按 `release.template.md` 更新 `release.md`，最后创建并推送 `v*` 标签；如果要重发同一版本，应删除失败 release 或改用新标签。
- **紧急程度**：低

### Bug 修复：Playwright install 参数透传
- **发现于**：`review-notes/code-review-2026-07-05-ci-playwright.md`
- **现象**：CI 中 `npm exec playwright install --with-deps chromium` 可能把 `--with-deps` 当作 npm 参数处理，干净 runner 上 Chromium 系统依赖安装不稳定。
- **根因**：`npm exec` 调用缺少 `--` 分隔符，工作流检查没有覆盖这类 shell 参数透传细节。
- **修复**：CI 改为 `npm --prefix web exec -- playwright install --with-deps chromium`，并新增 `scripts/check-workflows.sh` 持久检查。
- **回归测试**：`make check-workflows`、`make ci`。
- **为什么原测试没覆盖**：`actionlint` 只能校验 workflow 语法，不能识别 `npm exec` 与被执行命令参数之间的语义边界。
- **紧急程度**：中
