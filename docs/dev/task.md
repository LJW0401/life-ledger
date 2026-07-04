# life-ledger 开发任务计划

> 创建日期：2026-07-04
> 状态：已审核
> 版本：v1.0
> 关联需求文档：`docs/dev/prd.md`
> 关联架构文档：`docs/dev/sad.md`
> 关联测试文档：`docs/dev/tpd.md`

## 1. 计划边界

本文档把已审核的 PRD 和 SAD 拆成可执行开发任务。任务按阶段推进，每个阶段都必须产生可运行、可验证的增量。

不在本文档中重复展开完整 API 字段、SQLite DDL 或部署脚本细节；这些内容分别落到：

- `docs/dev/api-design.md`
- `docs/dev/data-model.md`
- `docs/dev/deployment.md`

## 2. 安全门控命令

项目脚手架建立后，以下命令是开发任务的基础门控。

后端门控：

```bash
test -z "$(gofmt -l ./cmd ./internal)"
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/server
```

前端门控：

```bash
npm run typecheck
npm run lint
npm run build
npm run test:e2e
```

发布前完整门控：

```bash
go vet ./... && go test ./... && npm run typecheck && npm run lint && npm run build
```

说明：在 `cmd/`、`internal/` 和 `web/` 尚未创建前，文档类任务以 `git diff --check` 作为临时门控；脚手架任务完成后，所有新增代码任务必须执行对应门控。

## 3. 阶段总览

| 阶段 | 名称 | 目标 | 主要需求 |
|------|------|------|----------|
| 0 | 详细设计补齐 | 补齐 API、数据模型、部署恢复文档 | R1-R19 |
| 1 | 工程骨架与运行基础 | 建立 Go + React 单体工程、配置、SQLite、内嵌前端 | R1-R3 |
| 2 | 认证与安全底座 | 实现单用户登录、设备会话、限速、CSRF、审计、安全头 | R10-R17 |
| 3 | 日期与标签纵向切片 | 完成重要日期和标签的后端、前端、测试闭环 | R4、R9 |
| 4 | 账单、预算与统计 | 完成账单、预算、基础统计和性能约束 | R5、R8、R19 |
| 5 | Excel 导入导出 | 完成账单模板、导入、导出和错误回滚 | R6 |
| 6 | 决策与备份部署 | 完成决策记录、备份命令、恢复说明和部署冒烟 | R7、R18 |
| 7 | 发布硬化 | 全量回归、安全回归、性能验证和发布准入 | R1-R19 |

## 4. 串行/并行执行拆分

### 4.1 拆分原则

- 阶段门控仍按串行推进：前置阶段的集成门控未通过时，不进入依赖它的下一批任务。
- 功能工作项与其 Smoke 测试、异常测试视为一个局部串行组；不能只做功能实现后把测试延后。
- 可以并行的任务必须满足：接口契约已确定、数据模型已确定、不会同时修改同一模块核心文件。
- 涉及 migration、认证 middleware、CSRF、审计和 Excel transaction 的任务默认串行，除非已拆出清晰接口边界。

### 4.2 串行主线

以下主线必须保持串行：

```text
阶段 0 文档契约
  -> 阶段 1 工程骨架、配置、SQLite、前端壳
  -> 阶段 2 认证、安全、审计底座
  -> 阶段 3/4/6 业务纵向切片并行开发
  -> 阶段 5 Excel 导入导出
  -> 阶段 7 发布硬化
```

必须串行的关键门控：

| 顺序 | 门控 | 解锁内容 |
|------|------|----------|
| 1 | WI-0.4 文档一致性检查 | 工程脚手架和契约实现 |
| 2 | WI-1.7 工程骨架与配置集成 | SQLite、前端壳、安全底座继续开发 |
| 3 | WI-1.14 运行基础集成 | 认证、安全和业务模块开发 |
| 4 | WI-2.7 认证基础集成 | CSRF、安全头、审计继续开发 |
| 5 | WI-2.14 安全底座集成 | 所有业务 API 和业务页面开发 |
| 6 | WI-4.7 账单预算 API 集成 | Excel 导入事务写入 |
| 7 | WI-5.11 Excel 端到端集成 | 发布硬化 |
| 8 | WI-7.10 发布准入 | 首版发布 |

### 4.3 可并行任务组

| 并行组 | 前置条件 | 可并行工作 | 汇合门控 |
|--------|----------|------------|----------|
| P0 文档契约 | 无 | WI-0.1 数据模型、WI-0.2 API 设计、WI-0.3 部署恢复文档 | WI-0.4 |
| P1 运行基础 | WI-1.1 完成 | A: WI-1.4~WI-1.10 配置与 SQLite；B: WI-1.11~WI-1.13 前端壳与路由 | WI-1.14 |
| P2 安全底座后半段 | WI-2.7 完成 | A: WI-2.8~WI-2.10 CSRF/CORS/安全头；B: WI-2.11~WI-2.13 审计和脱敏 | WI-2.14 |
| P3 业务后端切片 | WI-2.14 完成 | A: WI-3.1~WI-3.7 标签与日期；B: WI-4.1~WI-4.7 账单、预算、统计 API；C: WI-6.1~WI-6.3 决策 API | 各自阶段门控，最终进入 WI-6.11 |
| P4 业务前端切片 | 对应 API 契约稳定 | A: WI-3.4~WI-3.6 日期页；B: WI-4.8~WI-4.10 账单页；C: WI-6.4~WI-6.6 决策页 | WI-3.7、WI-4.12、WI-6.7 |
| P5 备份部署 | WI-1.14 完成，`deployment.md` 已定稿 | WI-6.8~WI-6.10 可与决策页开发并行 | WI-6.11 |
| P6 发布硬化准备 | WI-5.11 和 WI-6.11 完成 | A: WI-7.1~WI-7.3 安全回归；B: WI-7.4~WI-7.6 性能回归；C: WI-7.7~WI-7.9 发布包和文档 | WI-7.10 |

### 4.4 不应并行的任务

| 任务范围 | 原因 |
|----------|------|
| migration 设计与 migration 实现 | 并行修改 schema 容易造成 migration 顺序和回滚语义冲突。 |
| 认证 Cookie、设备会话、CSRF middleware | 共享登录态和请求安全边界，必须先形成稳定行为。 |
| 业务功能与对应异常测试 | 测试交织规则要求功能后立即补 Smoke 和异常测试。 |
| Excel 导入校验与 transaction 写入 | 导入失败不能部分写入，校验和写入语义必须串行收敛。 |
| 发布准入前的最后修复 | 任何修复必须重新跑对应门控，不能绕过 WI-7.10。 |

### 4.5 推荐执行批次

| 批次 | 类型 | 工作项 | 说明 |
|------|------|--------|------|
| B0 | 并行 | WI-0.1、WI-0.2、WI-0.3 | 三份详细设计文档可由不同上下文并行起草。 |
| B1 | 串行 | WI-0.4 | 统一术语、接口、字段和部署约束。 |
| B2 | 串行 | WI-1.1~WI-1.3 | 先建立可运行工程骨架和最小测试。 |
| B3 | 并行 | WI-1.4~WI-1.10、WI-1.11~WI-1.13 | 配置/SQLite 与前端壳可并行，最后统一集成。 |
| B4 | 串行 | WI-1.14、WI-2.1~WI-2.7 | 安全底座前半段必须先稳定登录和设备会话。 |
| B5 | 并行 | WI-2.8~WI-2.10、WI-2.11~WI-2.13 | CSRF/安全头和审计脱敏可并行，最后统一安全门控。 |
| B6 | 并行 | WI-3.1~WI-3.7、WI-4.1~WI-4.7、WI-6.1~WI-6.3 | 安全底座完成后，三个业务后端切片可并行。 |
| B7 | 并行 | WI-3.4~WI-3.6、WI-4.8~WI-4.10、WI-6.4~WI-6.6 | 对应 API 稳定后，三个业务页面可并行。 |
| B8 | 串行 | WI-5.1~WI-5.7 | Excel 依赖账单 API 和 transaction 语义，先做后端闭环。 |
| B9 | 并行 | WI-5.8~WI-5.11、WI-6.8~WI-6.11 | Excel UI 与备份部署可并行，最后做端到端集成。 |
| B10 | 并行 | WI-7.1~WI-7.9 | 安全、性能、发布包三条硬化线可并行。 |
| B11 | 串行 | WI-7.10 | 最终发布准入必须单点收敛。 |

## 5. 工作项

### 阶段 0：详细设计补齐

**目标**：把后续编码依赖的 API、数据模型和部署恢复细节先固化，降低实现阶段返工。

#### WI-0.1 [S] 编写数据模型设计文档

- **状态**：已完成
- **描述**：创建 `docs/dev/data-model.md`，定义 SQLite 表、字段、索引、migration 顺序和约束。
- **验收标准**：
  1. 覆盖 PRD 数据对象：重要日期、账单、预算、决策、标签、设备会话、登录失败、审计、备份元数据。
  2. 包含 SAD 要求的查询索引。
  3. 临时门控：`git diff --check` 通过。
- **Notes**：
  - Pattern：先文档化 schema，再实现 migration。
  - Reference：`docs/dev/prd.md`、`docs/dev/sad.md`。
  - Hook point：`internal/db/migrations/`。

#### WI-0.2 [S] 编写 API 设计文档

- **状态**：已完成
- **描述**：创建 `docs/dev/api-design.md`，定义 REST 路由、请求体、响应体、错误码和认证/CSRF 约定。
- **验收标准**：
  1. 覆盖 SAD 中列出的认证、日期、账单、预算、Excel、决策 API。
  2. 明确 `csrf_token` 由登录响应和 `GET /api/session` 返回。
  3. 临时门控：`git diff --check` 通过。
- **Notes**：
  - Pattern：接口先行，前后端按同一契约实现。
  - Reference：SAD 第 5 章。
  - Hook point：`internal/api/`、`web/src/api/`。

#### WI-0.3 [S] 编写部署与恢复文档

- **状态**：已完成
- **描述**：创建 `docs/dev/deployment.md`，定义 Linux 部署目录、Caddy 反代、权限、备份和手动恢复流程。
- **验收标准**：
  1. 覆盖单二进制、`config.toml`、数据目录、Caddy、备份包、手动恢复。
  2. 明确文件权限：`config.toml=600`、数据目录 `700`、数据库 `600`。
  3. 临时门控：`git diff --check` 通过。
- **Notes**：
  - Pattern：部署文档作为发布冒烟依据。
  - Reference：PRD R18、SAD 第 7 章。
  - Hook point：`internal/backup/`、发布脚本。

#### WI-0.4 [集成门控] 文档一致性检查

- **状态**：已完成
- **描述**：校验 PRD、SAD、TPD、API、数据模型、部署文档之间的命名和约束一致。
- **验收标准**：
  1. 旧文件名、已移除范围和安全约束在文档中表达一致；允许在“排除项、禁止项、ADR 背景”中出现说明性文字。
  2. `git diff --check` 通过。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 开发者开始编码, When 查阅 `docs/dev/`, Then API、数据模型和部署恢复细节均有对应文档。
2. Given 后续实现发现接口或表结构变化, When 修改设计, Then 同步更新对应文档。
3. 阶段状态：已完成。

**完成日期**：2026-07-04
**验收结果**：通过
**安全门控**：`git diff --check` 通过
**集成门控**：WI-0.4 通过
**备注**：已补齐 API、数据模型、部署恢复三份详细设计文档，后续实现以这些契约为准。

---

### 阶段 1：工程骨架与运行基础

**目标**：建立可运行的单体工程，完成配置读取、SQLite 初始化、内嵌前端和基础路由。

#### WI-1.1 [M] 搭建 Go + React 单体工程骨架

- **描述**：创建 `cmd/server/`、`internal/`、`web/`、`go.mod`、前端 `package.json` 和构建脚本。
- **验收标准**：
  1. `go build ./cmd/server` 可以生成服务入口。
  2. `npm run build` 可以生成前端 dist。
  3. 安全门控：`go vet ./...`、`go test ./...`、`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：入口只做编排，依赖组装放 `internal/app/`。
  - Reference：SAD 第 6 章。
  - Hook point：`cmd/server/main.go`、`internal/app/`。

#### WI-1.2 [S] Smoke 测试 — 工程骨架

- **描述**：添加最小 E2E smoke，验证服务可启动并返回前端入口。
- **验收标准**：
  1. Given 服务启动, When 请求 `/`, Then 返回前端入口或重定向到 `/important-dates`。
  2. 安全门控：`go test ./...`、`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-1.3 [S] 异常测试 — 工程骨架

- **描述**：覆盖脚手架早期的启动失败和路由异常。
- **覆盖场景清单**：
  - [x] 失败依赖：前端 dist 缺失时构建失败，而不是运行时静默空白。
  - [x] 权限/认证：未实现认证前标记受保护路由测试为 pending，不伪造通过。
  - [x] 异常恢复：重复启动同端口时返回明确监听错误。
- **实现手段**：E2E 启动脚本 + 临时端口冲突。
- **断言目标**：进程非零退出或测试失败信息明确。
- **验收标准**：
  1. 上述异常场景有自动化覆盖或明确 pending 标记。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-1.3G [集成门控] 工程骨架最小集成

- **描述**：验证脚手架、最小服务启动和早期 E2E 测试可以一起运行。
- **验收标准**：
  1. `go test ./... && npm run test:e2e` 通过。
  2. 失败时阻止继续进入配置和数据库任务。
- **Notes**：
  - Pattern：局部门控，控制首个功能测试组的集成风险。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地门控命令。

#### WI-1.4 [M] 实现配置读取与权限检查

- **描述**：实现 `config.toml` 读取、默认值、必填项校验、敏感配置和文件权限检查。
- **验收标准**：
  1. 合法配置可以启动服务。
  2. 缺失 `username`、`password_hash`、`session_secret` 或密钥过短时启动失败。
  3. 安全门控：`go test ./...`、`go vet ./...` 通过。
- **Notes**：
  - Pattern：fail first，配置错误阻止启动。
  - Reference：PRD R1、R10、R15。
  - Hook point：`internal/config/`。

#### WI-1.5 [S] Smoke 测试 — 配置读取

- **描述**：验证合法 `config.toml` 能驱动服务监听本机地址。
- **验收标准**：
  1. Given 合法配置, When 运行 `./life-ledger`, Then 服务监听配置端口。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-1.6 [S] 异常测试 — 配置读取

- **描述**：覆盖配置非法、权限过宽和敏感字段泄露。
- **覆盖场景清单**：
  - [x] 非法输入：TOML 格式错误、必填字段缺失、明文密码字段。
  - [x] 边界值：`session_secret` 少于 32 字节。
  - [x] 权限/认证：`config.toml` 权限不是 `600`。
  - [x] 异常恢复：失败后修正配置再启动成功。
- **实现手段**：临时目录 + 测试配置文件。
- **断言目标**：非零退出、错误信息不包含敏感值。
- **验收标准**：
  1. 异常场景全部有自动化覆盖。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-1.7 [集成门控] 工程骨架与配置集成

- **描述**：验证脚手架、配置和最小服务启动集成状态。
- **验收标准**：
  1. `go vet ./... && go test ./... && npm run typecheck && npm run build` 通过。
  2. 服务默认不监听公网地址。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-1.8 [M] 实现 SQLite 初始化和内嵌 migration

- **描述**：实现 SQLite 打开、文件权限、内嵌 migration、`schema_migrations` 和 transaction helper。
- **验收标准**：
  1. 首次启动自动创建数据库并执行 migration。
  2. 已执行 migration 不重复执行。
  3. 安全门控：`go test ./...`、`go test -race ./...` 通过。
- **Notes**：
  - Pattern：repository 之后接入，当前先实现基础设施。
  - Reference：PRD R2、SAD 第 4 章。
  - Hook point：`internal/db/`、`internal/db/migrations/`。

#### WI-1.9 [S] Smoke 测试 — SQLite 初始化

- **描述**：验证临时数据库首次创建和重启读取。
- **验收标准**：
  1. Given 数据库不存在, When 启动服务, Then 数据库文件和 schema 被创建。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-1.10 [S] 异常测试 — SQLite 初始化

- **描述**：覆盖 migration 失败、权限错误和 transaction 回滚。
- **覆盖场景清单**：
  - [x] 失败依赖：数据库目录不可写。
  - [x] 非法输入：损坏或失败的 migration。
  - [x] 异常恢复：修复 migration 后可以重新启动。
  - [x] 异常恢复：transaction 内失败不产生部分写入。
- **实现手段**：临时目录权限切换 + 测试 migration fixture。
- **断言目标**：非零退出、失败 migration 编号、DB 状态未污染。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-1.10G [集成门控] 配置与数据库集成

- **描述**：验证配置读取、文件权限、SQLite 初始化和 migration 可以共同完成启动链路。
- **验收标准**：
  1. `go vet ./... && go test ./... && go test -race ./...` 通过。
  2. 合法配置能创建数据库，非法配置不会创建脏数据库。
- **Notes**：
  - Pattern：局部门控，收敛配置和数据库启动路径。
  - Reference：PRD R1、R2、R15。
  - Hook point：`internal/config/`、`internal/db/`。

#### WI-1.11 [M] 实现内嵌前端和三页路由壳

- **描述**：实现 Go `embed` 静态资源服务、SPA fallback、React 三页导航壳。
- **验收标准**：
  1. `/` 自动进入 `/important-dates`。
  2. 刷新 `/important-dates`、`/transactions`、`/decisions` 不返回 404。
  3. 安全门控：`go test ./...`、`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：非 `/api/*` fallback 到 `index.html`。
  - Reference：PRD R3、UI overview。
  - Hook point：`internal/web/`、`web/src/`。

#### WI-1.12 [S] Smoke 测试 — 三页路由

- **描述**：用 E2E 验证三个一级页面可导航、刷新和高亮。
- **验收标准**：
  1. Given 用户访问三条路由, When 页面刷新, Then 仍展示对应页面。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-1.13 [S] 异常测试 — 三页路由

- **描述**：覆盖未知 API、未知页面和资源缺失。
- **覆盖场景清单**：
  - [x] 非法输入：未知 `/api/*` 返回 JSON 404。
  - [x] 非法输入：未知非 API 路由返回 SPA，不破坏前端。
  - [x] 失败依赖：静态资源缺失时构建门控失败。
- **实现手段**：Playwright + HTTP 请求断言。
- **断言目标**：状态码、content-type、页面可见内容。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...`、`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-1.14 [集成门控] 运行基础集成

- **描述**：验证阶段 1 全部运行基础可用。
- **验收标准**：
  1. 发布前完整门控除 `npm run lint` 外均已落地并通过；若 lint 脚本已创建则必须通过。
  2. 使用临时配置启动服务后，浏览器能访问三页壳。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 仓库完成阶段 1, When 运行本地服务, Then 用户可以看到三页空壳。
2. Given 配置或数据库异常, When 启动服务, Then 程序明确失败且不静默降级。
3. 阶段状态：未开始。

---

### 阶段 2：认证与安全底座

**目标**：先建立所有受保护功能依赖的安全基础，后续业务 API 默认挂载认证、CSRF、审计和安全响应头。

#### WI-2.1 [M] 实现单用户登录、退出和安全配置命令

- **描述**：实现 bcrypt 登录、HttpOnly Cookie、`hash-password`、`generate-secret` 和退出登录。
- **验收标准**：
  1. 正确凭据登录成功，错误凭据不创建登录态。
  2. 辅助命令不启动 HTTP、不修改数据库。
  3. 安全门控：`go test ./...`、`go test -race ./...` 通过。
- **Notes**：
  - Pattern：应用内认证，Caddy 不负责业务登录。
  - Reference：PRD R10、R13。
  - Hook point：`internal/auth/`、`cmd/server/`。

#### WI-2.2 [S] Smoke 测试 — 登录退出

- **描述**：覆盖登录页、登录成功、受保护页面访问和退出登录。
- **验收标准**：
  1. Given 用户输入正确凭据, When 登录, Then 进入业务页面。
  2. 安全门控：`npm run test:e2e`、`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-2.3 [S] 异常测试 — 登录和辅助命令

- **描述**：覆盖错误凭据、缺失配置、短密钥和辅助命令失败。
- **覆盖场景清单**：
  - [x] 非法输入：错误密码、空用户名、短 `session_secret`。
  - [x] 权限/认证：未登录访问页面和 API。
  - [x] 失败依赖：随机数源或 stdin 读取失败时命令非零退出。
- **实现手段**：API 测试 + CLI 测试。
- **断言目标**：401、非零退出、无登录态 Cookie。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-2.3G [集成门控] 登录退出集成

- **描述**：验证登录、退出、辅助命令和未登录拦截形成可用闭环。
- **验收标准**：
  1. `go test ./... && npm run test:e2e` 通过。
  2. 正确登录可进入页面，错误登录和未登录访问被拒绝。
- **Notes**：
  - Pattern：局部门控，先稳定认证入口再扩展设备会话。
  - Reference：PRD R10、R13。
  - Hook point：`internal/auth/`、`cmd/server/`。

#### WI-2.4 [M] 实现设备会话和登录失败限速

- **描述**：实现设备记录、session token 哈希、7 天固定过期、设备撤销、SQLite 持久化限速。
- **验收标准**：
  1. 登录成功创建设备会话，数据库只保存 token 哈希。
  2. 10 分钟内 5 次失败锁定 15 分钟，重启后仍生效。
  3. 安全门控：`go test ./...`、`go test -race ./...` 通过。
- **Notes**：
  - Pattern：token 原文只在 Cookie 中，服务端只存 hash。
  - Reference：PRD R11、R12。
  - Hook point：`internal/auth/`、`internal/db/`。

#### WI-2.5 [S] Smoke 测试 — 设备会话

- **描述**：验证同一设备 7 天内免登录、设备列表和撤销流程。
- **验收标准**：
  1. Given 有效 Cookie, When 访问页面, Then 不要求重新登录。
  2. 安全门控：`npm run test:e2e`、`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-2.6 [S] 异常测试 — 设备会话和限速

- **描述**：覆盖过期、撤销、不存在 token、锁定和重启恢复。
- **覆盖场景清单**：
  - [x] 边界值：会话刚好过期。
  - [x] 权限/认证：撤销设备、伪造 token、无 Cookie。
  - [x] 异常恢复：服务重启后锁定状态仍生效。
  - [x] 并发/竞态：重复撤销同一设备。
- **实现手段**：API 测试 + 可控时间源。
- **断言目标**：401、Cookie 清除、SQLite 状态正确。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-2.7 [集成门控] 认证基础集成

- **描述**：验证登录、设备、限速和受保护路由已经形成闭环。
- **验收标准**：
  1. `go test ./... && npm run test:e2e` 通过。
  2. 未登录用户无法访问三页和受保护 API。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-2.8 [M] 实现 CSRF、CORS、安全响应头和可信反代 IP

- **描述**：实现 CSRF token 获取与校验、基础安全头、禁止跨源 API、可信反代 IP 解析。
- **验收标准**：
  1. 登录响应和 `GET /api/session` 返回 `csrf_token`。
  2. 非 GET/HEAD/OPTIONS 写操作必须校验 `X-CSRF-Token`。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：安全 middleware 默认挂载到 API。
  - Reference：PRD R14、R17，SAD 第 5 章。
  - Hook point：`internal/security/`、`internal/api/`。

#### WI-2.9 [S] Smoke 测试 — CSRF 与安全头

- **描述**：验证正确 token 的写请求可通过，响应包含安全头。
- **验收标准**：
  1. Given 登录态和正确 token, When 发起写请求, Then 请求进入业务 handler。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-2.10 [S] 异常测试 — CSRF、CORS 和可信 IP

- **描述**：覆盖缺失 token、错误 token、跨域请求和伪造代理头。
- **覆盖场景清单**：
  - [x] 权限/认证：缺失、错误、过期 CSRF token。
  - [x] 非法输入：第三方 Origin 请求。
  - [x] 非法输入：非可信来源伪造 `X-Forwarded-For`。
  - [x] 边界值：GET/HEAD/OPTIONS 不要求 CSRF。
- **实现手段**：API 测试构造请求头。
- **断言目标**：403、无 `Access-Control-Allow-Origin: *`、解析 IP 正确。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-2.10G [集成门控] CSRF 与安全头集成

- **描述**：验证 CSRF、CORS、安全响应头和可信反代 IP 解析可以共同保护 API。
- **验收标准**：
  1. `go test ./...` 通过。
  2. 写请求无 token 返回 403，响应不包含 `Access-Control-Allow-Origin: *`。
- **Notes**：
  - Pattern：局部门控，收敛请求安全 middleware。
  - Reference：PRD R14、R17。
  - Hook point：`internal/security/`、`internal/api/`。

#### WI-2.11 [M] 实现审计事件和日志脱敏

- **描述**：实现 `audit_events` 写入、审计查询基础、日志脱敏工具和安全事件调用点。
- **验收标准**：
  1. 登录、退出、设备创建、设备撤销、配置安全失败写入审计。
  2. 日志和审计不记录敏感值或完整业务内容。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：审计写入是事实来源，日志只是脱敏摘要。
  - Reference：PRD R16。
  - Hook point：`internal/audit/`。

#### WI-2.12 [S] Smoke 测试 — 审计事件

- **描述**：验证登录、退出、设备撤销能写入 `audit_events`。
- **验收标准**：
  1. Given 发生安全事件, When 查询审计表, Then 存在脱敏结构化事件。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-2.13 [S] 异常测试 — 审计和脱敏

- **描述**：覆盖数据库不可用前的启动失败、敏感值泄露和删除事件脱敏。
- **覆盖场景清单**：
  - [x] 失败依赖：数据库尚不可用时配置权限检查失败。
  - [x] 非法输入：日志消息包含敏感字段。
  - [x] 权限/认证：未登录请求不产生业务审计污染。
- **实现手段**：日志捕获 + 临时权限测试。
- **断言目标**：stderr 脱敏、非零退出、审计字段不含敏感值。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-2.14 [集成门控] 安全底座集成

- **描述**：验证认证、设备、限速、CSRF、安全头、可信 IP 和审计集成状态。
- **验收标准**：
  1. `go test ./... && go test -race ./... && npm run test:e2e` 通过。
  2. 安全回归中认证、CSRF、CORS、响应头、审计、设备撤销和限速全部通过。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 未登录用户访问业务页面或 API, When 请求到达服务端, Then 被登录页或 401 拦截。
2. Given 已登录用户发起写请求, When 缺失 CSRF token, Then 返回 403 且业务数据不变。
3. 阶段状态：未开始。

---

### 阶段 3：日期与标签纵向切片

**目标**：完成第一个完整业务纵向切片，打通 SQLite、API、前端、E2E 和审计。

#### WI-3.1 [M] 实现标签服务和通用筛选基础

- **描述**：实现 `tags`、`entity_tags` repository、标签关联和按标签筛选。
- **验收标准**：
  1. 标签可被重要日期、账单、决策复用。
  2. 按标签筛选只返回关联记录。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：标签作为跨业务对象的多态关联。
  - Reference：PRD R9、SAD 数据模型。
  - Hook point：`internal/domain/tags/`。

#### WI-3.2 [S] Smoke 测试 — 标签

- **描述**：验证创建标签、关联记录、按标签筛选。
- **验收标准**：
  1. Given 记录关联标签, When 按标签查询, Then 返回对应记录。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-3.3 [S] 异常测试 — 标签

- **描述**：覆盖空标签、重复标签、非法对象类型和删除关联。
- **覆盖场景清单**：
  - [x] 非法输入：空标签、超长标签、非法对象类型。
  - [x] 并发/竞态：重复创建同名标签。
  - [x] 异常恢复：删除业务记录后关联不破坏查询。
- **实现手段**：API/集成测试。
- **断言目标**：400、唯一性、关联清理正确。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-3.3G [集成门控] 标签服务集成

- **描述**：验证标签字典、对象关联和筛选基础可以被业务模块复用。
- **验收标准**：
  1. `go test ./...` 通过。
  2. 标签创建、重复处理、关联查询和删除关联均通过。
- **Notes**：
  - Pattern：局部门控，先稳定跨业务复用能力。
  - Reference：PRD R9。
  - Hook point：`internal/domain/tags/`。

#### WI-3.4 [M] 实现重要日期 API 和页面

- **描述**：实现重要日期 CRUD、重复规则校验、列表排序、前端日期页表单和列表。
- **验收标准**：
  1. 用户可以新增、查看、编辑、删除重要日期。
  2. 重复规则支持不重复、每年、每月、每周。
  3. 安全门控：`go test ./...`、`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：先 API，再前端接入。
  - Reference：PRD R4、UI 日期页文档。
  - Hook point：`internal/domain/importantdates/`、`web/src/`。

#### WI-3.5 [S] Smoke 测试 — 重要日期

- **描述**：用 E2E 验证日期页新增、编辑、删除和标签展示。
- **验收标准**：
  1. Given 登录用户打开日期页, When 创建日期记录, Then 列表展示新记录。
  2. 安全门控：`npm run test:e2e`、`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-3.6 [S] 异常测试 — 重要日期

- **描述**：覆盖必填缺失、非法重复规则、未登录、删除不存在记录。
- **覆盖场景清单**：
  - [x] 非法输入：标题缺失、日期格式错误、重复规则非法。
  - [x] 权限/认证：未登录访问 CRUD API。
  - [x] 边界值：删除不存在 ID。
- **实现手段**：API 测试 + Playwright 表单校验。
- **断言目标**：400、401、404、UI 错误提示。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./... && npm run test:e2e` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-3.7 [集成门控] 日期与标签切片集成

- **描述**：验证标签和重要日期从数据库到 UI 的闭环。
- **验收标准**：
  1. `go test ./... && npm run typecheck && npm run test:e2e` 通过。
  2. 删除日期记录写入审计事件。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 登录用户访问日期页, When 完成新增、编辑、删除, Then SQLite、API 和 UI 状态一致。
2. Given 日期表单输入非法, When 保存, Then 显示明确错误且不写入数据库。
3. 阶段状态：未开始。

---

### 阶段 4：账单、预算与统计

**目标**：完成账单、预算和基础统计，形成主要业务数据闭环。

#### WI-4.1 [M] 实现账单 API 和数据层

- **描述**：实现账单 CRUD、筛选、分页、收支统计字段和审计调用。
- **验收标准**：
  1. 支持日期、时间、类型、金额、分类、计入收支、计入预算、所属账本等字段。
  2. 列表默认分页，最大 `page_size=200`。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：repository 封装 SQL，service 承担业务规则。
  - Reference：PRD R5、SAD 查询性能策略。
  - Hook point：`internal/domain/transactions/`。

#### WI-4.2 [S] Smoke 测试 — 账单 API

- **描述**：验证账单新增、列表筛选、编辑、删除。
- **验收标准**：
  1. Given 合法账单, When 保存, Then 列表和详情可读取。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-4.3 [S] 异常测试 — 账单 API

- **描述**：覆盖金额、枚举、缺字段、分页和认证异常。
- **覆盖场景清单**：
  - [x] 非法输入：金额为空、为零、格式非法，类型非法。
  - [x] 边界值：`page_size` 超过 200。
  - [x] 权限/认证：未登录或缺 CSRF 写入。
  - [x] 异常恢复：删除不存在记录。
- **实现手段**：API 测试。
- **断言目标**：400、401、403、404、DB 不变。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-4.3G [集成门控] 账单 API 集成

- **描述**：验证账单 CRUD、筛选、分页、认证和审计基础集成状态。
- **验收标准**：
  1. `go test ./... && go test -race ./...` 通过。
  2. 账单新增、筛选、删除和未登录拦截均通过。
- **Notes**：
  - Pattern：局部门控，先稳定账单 API 再接预算统计。
  - Reference：PRD R5。
  - Hook point：`internal/domain/transactions/`、`internal/api/`。

#### WI-4.4 [M] 实现预算和基础统计

- **描述**：实现月份 + 分类预算、预算消耗、收入支出余额和分类占比。
- **验收标准**：
  1. 支出且 `计入预算=是` 才消耗预算。
  2. `计入收支=否` 不参与收入、支出、余额统计。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：实时聚合查询，不落冗余统计表。
  - Reference：PRD R8、R19。
  - Hook point：`internal/domain/budgets/`、`transactions` 查询。

#### WI-4.5 [S] Smoke 测试 — 预算和统计

- **描述**：验证预算创建、账单计入预算、统计卡片返回正确结果。
- **验收标准**：
  1. Given 月份分类预算和支出账单, When 查询预算, Then 返回已用、剩余、比例。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-4.6 [S] 异常测试 — 预算和统计

- **描述**：覆盖非法预算金额、收入账单、计入预算否、筛选边界。
- **覆盖场景清单**：
  - [x] 非法输入：预算金额为空、为零、格式非法。
  - [x] 边界值：超支、无预算、无账单。
  - [x] 权限/认证：未登录或缺 CSRF。
  - [x] 异常恢复：修改或删除参与预算账单后统计重算。
- **实现手段**：API/集成测试。
- **断言目标**：400、统计值正确、DB 状态正确。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-4.7 [集成门控] 账单预算 API 集成

- **描述**：验证账单、预算、统计和审计集成状态。
- **验收标准**：
  1. `go test ./... && go test -race ./...` 通过。
  2. 删除账单写入审计事件。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-4.8 [M] 实现账单页前端

- **描述**：实现账单列表、表单、筛选、统计卡片、预算使用展示。
- **验收标准**：
  1. 用户可以在账单页完成账单新增、筛选、编辑、删除。
  2. 页面展示收入、支出、余额和预算使用情况。
  3. 安全门控：`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：页面内部局部切换，不新增一级页面。
  - Reference：UI 账单页文档。
  - Hook point：`web/src/pages/transactions`。

#### WI-4.9 [S] Smoke 测试 — 账单页

- **描述**：用 E2E 验证账单页新增、筛选、统计和预算展示。
- **验收标准**：
  1. Given 登录用户创建账单, When 返回列表, Then 统计卡片同步更新。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-4.10 [S] 异常测试 — 账单页

- **描述**：覆盖表单校验、API 错误提示和无数据状态。
- **覆盖场景清单**：
  - [x] 非法输入：金额为空或格式非法。
  - [x] 权限/认证：登录过期后保存失败。
  - [x] 失败依赖：API 返回 500 时页面展示错误。
  - [x] 边界值：空列表和超支预算。
- **实现手段**：Playwright + API route mock。
- **断言目标**：UI 错误提示、无重复写入、布局不崩。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-4.11 [S] 性能测试 — 账单和统计

- **描述**：验证 5000 条以内账单列表和统计满足 P95 小于 300ms 的目标。
- **验收标准**：
  1. Given 测试数据库有 5000 条账单, When 请求常规列表和统计 API, Then P95 小于 300ms。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：纯文档或发布收口任务，按文档一致性门控验收。
  - Reference：`docs/dev/prd.md`、`docs/dev/sad.md`、`docs/dev/tpd.md`。
  - Hook point：`docs/dev/` 文档集。
#### WI-4.12 [集成门控] 账单页完整切片

- **描述**：验证账单、预算、统计从 UI 到 SQLite 全链路。
- **验收标准**：
  1. `go test ./... && npm run typecheck && npm run test:e2e` 通过。
  2. 账单核心路径满足 TPD 发布准入。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 登录用户使用账单页, When 完成账单和预算操作, Then 统计与预算实时反映最新数据。
2. Given 非法账单或预算输入, When 保存, Then API 和 UI 均给出明确错误且不写入脏数据。
3. 阶段状态：未开始。

---

### 阶段 5：Excel 导入导出

**目标**：完成账单 Excel 交换能力，保证失败不部分写入、错误不泄露原始内容。

#### WI-5.1 [M] 实现 Excel 模板和导出

- **描述**：使用 excelize 实现模板下载和按筛选条件导出 `.xlsx`。
- **验收标准**：
  1. 模板包含 PRD 指定必填和选填列表头。
  2. 导出文件可被 Excel 打开。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：Excel 只作为交换格式。
  - Reference：PRD R6。
  - Hook point：`internal/excel/`。

#### WI-5.2 [S] Smoke 测试 — Excel 模板和导出

- **描述**：验证模板和导出文件格式、表头、筛选结果。
- **验收标准**：
  1. Given 用户请求模板, When 打开文件, Then 表头和字段顺序正确。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-5.3 [S] 异常测试 — Excel 模板和导出

- **描述**：覆盖未登录、空数据、非法筛选和文件生成错误。
- **覆盖场景清单**：
  - [x] 权限/认证：未登录下载导出。
  - [x] 边界值：空账单导出。
  - [x] 非法输入：非法筛选参数。
  - [x] 失败依赖：写响应失败时不写入审计脏数据。
- **实现手段**：API 测试 + response writer fault fixture。
- **断言目标**：401、400、文件可打开、日志脱敏。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-5.3G [集成门控] Excel 导出集成

- **描述**：验证模板下载、导出和认证保护在账单数据上可用。
- **验收标准**：
  1. `go test ./...` 通过。
  2. 模板和导出文件均可打开，未登录导出被拒绝。
- **Notes**：
  - Pattern：局部门控，先稳定只读 Excel 能力。
  - Reference：PRD R6、TPD 第 7 章。
  - Hook point：`internal/excel/`、`internal/api/`。

#### WI-5.4 [M] 实现 Excel 导入校验和事务写入

- **描述**：实现 `.xlsx` 接收、大小限制、首个工作表解析、整表校验、transaction 写入和审计。
- **验收标准**：
  1. 合法 5000 行以内文件可以导入。
  2. 任一错误行导致不写入任何记录。
  3. 安全门控：`go test ./...`、`go test -race ./...` 通过。
- **Notes**：
  - Pattern：先校验整表，再事务写入。
  - Reference：TPD 第 7 章。
  - Hook point：`internal/excel/`、`transactions` service。

#### WI-5.5 [S] Smoke 测试 — Excel 导入

- **描述**：验证合法 `.xlsx` 文件导入账单并写入审计。
- **验收标准**：
  1. Given 合法导入文件, When 导入, Then 账单全部写入且审计记录成功事件。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-5.6 [S] 异常测试 — Excel 导入

- **描述**：覆盖格式、大小、行数、表头、必填、枚举、金额和回滚。
- **覆盖场景清单**：
  - [x] 非法输入：非 `.xlsx`、错误表头、必填列空、枚举非法、金额非法。
  - [x] 边界值：超过 5MB、超过 5000 行、多工作表只读第一个。
  - [x] 异常恢复：错误行导致 transaction 回滚。
  - [x] 权限/认证：未登录或缺 CSRF。
- **实现手段**：fixture 文件 + API 测试。
- **断言目标**：400、行号列名原因、DB 无部分写入、不回显整行原始内容。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-5.7 [集成门控] Excel 后端集成

- **描述**：验证 Excel 模板、导出、导入、审计和账单服务集成。
- **验收标准**：
  1. `go test ./... && go test -race ./...` 通过。
  2. 导入失败不产生部分写入。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-5.8 [M] 实现账单页 Excel UI

- **描述**：在账单页接入模板下载、导入上传、错误明细展示、导出按钮。
- **验收标准**：
  1. 用户可以在账单页下载模板、上传 Excel、查看错误明细、导出账单。
  2. 错误明细只展示行号、列名和原因。
  3. 安全门控：`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：导入错误作为结构化列表展示。
  - Reference：UI 账单页文档、TPD 第 7 章。
  - Hook point：`web/src/pages/transactions`。

#### WI-5.9 [S] Smoke 测试 — Excel UI

- **描述**：用 E2E 验证模板下载、合法文件导入和导出。
- **验收标准**：
  1. Given 登录用户上传合法文件, When 导入成功, Then 账单列表出现导入记录。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-5.10 [S] 异常测试 — Excel UI

- **描述**：覆盖非法文件、错误行展示、超大文件和登录过期。
- **覆盖场景清单**：
  - [x] 非法输入：非 `.xlsx`、错误表头、错误行。
  - [x] 边界值：超过大小限制。
  - [x] 权限/认证：登录过期上传。
  - [x] 失败依赖：API 返回 500。
- **实现手段**：Playwright file upload + API route mock。
- **断言目标**：UI 错误明细、不显示原始整行内容。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-5.11 [集成门控] Excel 端到端集成

- **描述**：验证账单页 Excel 从 UI 到 SQLite 的完整链路。
- **验收标准**：
  1. `go test ./... && npm run test:e2e` 通过。
  2. Excel 导入失败场景验证不能产生部分写入。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 用户上传合法 Excel, When 确认导入, Then 所有记录一次性写入数据库。
2. Given 用户上传包含错误行的 Excel, When 导入失败, Then 返回具体错误且数据库不写入任何记录。
3. 阶段状态：未开始。

---

### 阶段 6：决策与备份部署

**目标**：完成决策页核心功能、备份命令和部署恢复闭环。

#### WI-6.1 [M] 实现决策 API 和数据层

- **描述**：实现决策 CRUD、候选方案结构化保存、状态分组、复盘归档和标签关联。
- **验收标准**：
  1. 支持进行中、待复盘、已归档状态。
  2. 候选方案包含方案名称、优点、缺点和备注。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：决策主体和候选方案拆表。
  - Reference：PRD R7。
  - Hook point：`internal/domain/decisions/`。

#### WI-6.2 [S] Smoke 测试 — 决策 API

- **描述**：验证决策新增、候选方案、状态分组、复盘归档。
- **验收标准**：
  1. Given 决策设置复盘日期且日期到达, When 查询列表, Then 出现在待复盘分组。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-6.3 [S] 异常测试 — 决策 API

- **描述**：覆盖缺字段、非法状态、候选方案格式错误和未登录。
- **覆盖场景清单**：
  - [x] 非法输入：标题缺失、非法状态、候选方案字段缺失。
  - [x] 权限/认证：未登录或缺 CSRF。
  - [x] 边界值：删除不存在决策。
  - [x] 异常恢复：删除成功写入审计，不记录完整业务内容。
- **实现手段**：API/集成测试。
- **断言目标**：400、401、403、404、审计脱敏。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-6.3G [集成门控] 决策 API 集成

- **描述**：验证决策 CRUD、候选方案、状态分组、标签和审计基础集成。
- **验收标准**：
  1. `go test ./...` 通过。
  2. 决策创建、状态查询、删除审计和未登录拦截均通过。
- **Notes**：
  - Pattern：局部门控，先稳定决策后端再接页面。
  - Reference：PRD R7。
  - Hook point：`internal/domain/decisions/`、`internal/api/`。

#### WI-6.4 [M] 实现决策页前端

- **描述**：实现决策列表、详情、候选方案编辑、状态分组、复盘归档。
- **验收标准**：
  1. 用户可以在决策页完成新增、编辑、复盘和归档。
  2. 页面展示进行中、待复盘、已归档分组。
  3. 安全门控：`npm run typecheck`、`npm run build` 通过。
- **Notes**：
  - Pattern：结构化候选方案输入，不使用自由文本混合字段。
  - Reference：UI 决策页文档。
  - Hook point：`web/src/pages/decisions`。

#### WI-6.5 [S] Smoke 测试 — 决策页

- **描述**：用 E2E 验证决策创建、候选方案、待复盘和归档。
- **验收标准**：
  1. Given 登录用户创建决策并完成复盘, When 保存, Then 记录进入已归档分组。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-6.6 [S] 异常测试 — 决策页

- **描述**：覆盖表单缺失、API 错误、登录过期和状态切换异常。
- **覆盖场景清单**：
  - [x] 非法输入：标题缺失、候选方案缺字段。
  - [x] 权限/认证：登录过期保存。
  - [x] 失败依赖：API 返回 500。
  - [x] 边界值：空列表和无候选方案。
- **实现手段**：Playwright + API route mock。
- **断言目标**：UI 错误提示、无重复写入。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`npm run test:e2e` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-6.7 [集成门控] 决策切片集成

- **描述**：验证决策 API、UI、标签和审计集成。
- **验收标准**：
  1. `go test ./... && npm run test:e2e` 通过。
  2. 删除决策写入审计事件。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-6.8 [M] 实现备份命令和部署冒烟脚本

- **描述**：实现 `./life-ledger backup`，生成包含 SQLite、`config.toml`、`backup-meta.json` 的备份包，并补部署冒烟脚本。
- **验收标准**：
  1. 备份成功生成完整备份包。
  2. 备份命令不启动 HTTP、不修改数据库。
  3. 安全门控：`go test ./...`、`go build ./cmd/server` 通过。
- **Notes**：
  - Pattern：CLI 一次性命令和 HTTP 服务启动路径隔离。
  - Reference：PRD R18、deployment 文档。
  - Hook point：`internal/backup/`、`cmd/server/`。

#### WI-6.9 [S] Smoke 测试 — 备份和恢复

- **描述**：验证备份包内容和按文档恢复后的数据可读。
- **验收标准**：
  1. Given 已有数据库, When 执行 backup, Then 备份包包含 DB、配置和元数据。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-6.10 [S] 异常测试 — 备份和部署

- **描述**：覆盖无法读取数据库、无法读取配置、权限错误和恢复权限。
- **覆盖场景清单**：
  - [x] 失败依赖：数据库文件不可读、配置文件不可读。
  - [x] 权限/认证：备份包包含登录能力，需要文档提示。
  - [x] 异常恢复：恢复后权限不正确时服务启动失败。
- **实现手段**：临时目录权限切换 + CLI 测试。
- **断言目标**：非零退出、不启动 HTTP、数据库不变。
- **验收标准**：
  1. 异常场景全部通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-6.11 [集成门控] 决策、备份和部署集成

- **描述**：验证决策功能和备份恢复不会互相破坏。
- **验收标准**：
  1. `go test ./... && npm run test:e2e && go build ./cmd/server` 通过。
  2. 恢复演练后日期、账单、决策数据均可读。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 用户使用决策页, When 完成决策记录和复盘归档, Then 数据可持久化并可恢复。
2. Given 用户执行备份和恢复, When 重新启动服务, Then 业务数据和登录配置按文档恢复。
3. 阶段状态：未开始。

---

### 阶段 7：发布硬化

**目标**：完成全量测试、安全回归、性能验证、文档收口和发布准入。

#### WI-7.1 [M] 全量安全回归

- **描述**：按 TPD 第 8 章执行认证、设备、限速、CSRF、CORS、响应头、可信 IP、日志和审计回归。
- **验收标准**：
  1. 安全测试中认证、CSRF、CORS、响应头、日志脱敏、审计、设备撤销和登录失败限速全部通过。
  2. 安全门控：`go test ./... && npm run test:e2e` 通过。
- **Notes**：
  - Pattern：先自动化回归，再人工检查部署项。
  - Reference：`docs/dev/tpd.md`。
  - Hook point：安全测试套件。

#### WI-7.2 [S] 异常测试 — 发布安全边界

- **描述**：补齐发布前安全边界测试。
- **覆盖场景清单**：
  - [x] 权限/认证：所有受保护写接口无 token 拒绝。
  - [x] 非法输入：跨域请求和伪造代理头。
  - [x] 失败依赖：数据库不可用时启动失败。
  - [x] 异常恢复：修复权限和数据库后服务恢复。
- **实现手段**：API 测试 + CLI 启动测试。
- **断言目标**：401/403、非零退出、日志脱敏。
- **验收标准**：
  1. 所有场景通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-7.3 [集成门控] 安全发布门控

- **描述**：确认安全测试结果达到发布准入。
- **验收标准**：
  1. `go test ./... && go test -race ./... && npm run test:e2e` 通过。
  2. 日志和审计抽样检查不包含敏感值。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-7.4 [M] 性能测试 — 数据量回归

- **描述**：用 5000 条测试数据验证账单列表、统计、预算和 Excel 导入性能。
- **验收标准**：
  1. 常规 API 请求 P95 小于 300ms。
  2. Excel 导入 5000 行以内给出明确成功或失败结果。
  3. 安全门控：`go test ./...` 通过。
- **Notes**：
  - Pattern：构造固定 fixture，测试可重复。
  - Reference：SAD 查询性能策略、TPD 第 10 章。
  - Hook point：性能测试套件。

#### WI-7.5 [S] 异常测试 — 性能边界

- **描述**：覆盖分页上限、超大导入和索引缺失回归。
- **覆盖场景清单**：
  - [x] 边界值：`page_size` 超过 200。
  - [x] 边界值：Excel 超过 5000 行或 5MB。
  - [x] 异常恢复：性能测试数据清理后不污染开发库。
- **实现手段**：API 测试 + fixture 数据库。
- **断言目标**：400 或截断策略符合 API 文档、测试库清理。
- **验收标准**：
  1. 所有场景通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-7.6 [集成门控] 性能发布门控

- **描述**：确认性能和数据量测试达到发布准入。
- **验收标准**：
  1. 性能回归报告记录关键接口结果。
  2. `go test ./...` 通过。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
#### WI-7.7 [M] 文档和发布包收口

- **描述**：更新 README、部署文档、配置示例、恢复说明和发布检查清单。
- **验收标准**：
  1. README 能指导用户构建、配置、启动和备份。
  2. `config.example.toml` 不包含真实密钥。
  3. 安全门控：`git diff --check`、`go build ./cmd/server` 通过。
- **Notes**：
  - Pattern：发布文档必须与实际命令一致。
  - Reference：PRD 成功标准、deployment 文档。
  - Hook point：README、配置示例、发布目录。

#### WI-7.8 [S] Smoke 测试 — 发布包

- **描述**：构建发布二进制并在临时目录用示例配置启动。
- **验收标准**：
  1. Given 构建产物和配置文件, When 在临时目录启动, Then 浏览器可访问登录页。
  2. 安全门控：`go build ./cmd/server` 通过。

- **Notes**：
  - Pattern：端到端正常路径验证，断言用户可观测结果。
  - Reference：`docs/dev/tpd.md` 对应 Smoke 测试范围。
  - Hook point：E2E/API 测试套件。
#### WI-7.9 [S] 异常测试 — 发布包

- **描述**：覆盖缺少配置、权限错误、缺少数据目录和端口占用。
- **覆盖场景清单**：
  - [x] 非法输入：缺失配置、非法 TOML。
  - [x] 失败依赖：端口占用、数据目录不可写。
  - [x] 权限/认证：配置和数据库权限过宽。
  - [x] 异常恢复：修复问题后可启动。
- **实现手段**：CLI 启动测试 + 临时目录。
- **断言目标**：非零退出、错误信息明确、敏感值不泄露。
- **验收标准**：
  1. 所有场景通过。
  2. 安全门控：`go test ./...` 通过。

- **Notes**：
  - Pattern：从系统边界构造异常，不修改被测代码。
  - Reference：`docs/dev/tpd.md` 异常/边界测试要求。
  - Hook point：API/E2E/CLI 测试夹具。
#### WI-7.10 [集成门控] 发布准入

- **描述**：执行发布前完整门控，确认首版可交付。
- **验收标准**：
  1. `go vet ./... && go test ./... && npm run typecheck && npm run lint && npm run build` 通过。
  2. `npm run test:e2e` 通过。
  3. 备份恢复完成一次手工演练。

- **Notes**：
  - Pattern：阶段性汇合检查，失败则不进入后续任务。
  - Reference：本文件第 4 章串行/并行执行拆分。
  - Hook point：CI 门控/本地完整门控命令。
**阶段验收标准**：

1. Given 发布候选版本, When 执行完整门控, Then 所有测试和构建通过。
2. Given 用户按部署文档部署, When Caddy 反代访问服务, Then 可登录并操作三类核心页面。
3. 阶段状态：未开始。

## 6. 风险与应对

| 风险 | 影响 | 概率 | 应对措施 |
|------|------|------|----------|
| 安全功能开发晚于业务功能 | 后期重构成本高 | 中 | 阶段 2 先完成认证、CSRF、审计和安全头，后续业务默认挂载。 |
| Excel 导入污染数据 | 用户账单数据错误 | 中 | 阶段 5 强制先校验整表，再 transaction 写入，并把失败回滚作为门控。 |
| 前后端接口漂移 | E2E 大量失败 | 中 | 阶段 0 先写 API 文档，业务阶段按接口契约实现。 |
| SQLite 查询性能退化 | 账单页等待明显 | 中 | 阶段 4 加索引和分页，阶段 7 做 5000 条性能回归。 |
| 部署恢复文档与实现不一致 | 云服务器恢复失败 | 中 | 阶段 6 和 7 均安排备份恢复演练。 |
| 测试门控尚未落地 | 任务验收不可执行 | 中 | 阶段 1 建立脚本后，后续所有代码任务必须执行门控。 |
| 并行开发修改同一契约 | API、数据模型或测试夹具冲突 | 中 | 并行组进入汇合门控前先做文档和接口一致性检查。 |

## 7. 开发规范

### 7.1 代码规范

- Go 代码按 `gofmt` 格式化。
- 入口层只做编排，业务规则放领域服务，SQL 放 repository。
- 错误必须显式返回，不添加静默 fallback。
- 不在日志中输出敏感值或 Excel 原始内容。

### 7.2 测试规范

- 功能工作项后必须紧跟 Smoke 测试和异常/边界测试。
- 异常测试只从系统边界外注入异常，不修改被测代码。
- 失败断言必须检查可观测产物：HTTP 状态码、错误码、日志、DB 状态或 UI 提示。
- 集成门控失败时不进入下一组业务工作项。

### 7.3 Git 规范

- 每个阶段或可独立验收的任务组提交一次。
- 提交信息使用中文，描述为什么做这次变更。
- 不提交真实配置、真实数据库、真实 Excel 账单或密钥。

### 7.4 文档规范

- API、数据模型或部署行为变化时，必须同步更新 `docs/dev/` 对应文档。
- UI 纯表现和交互细节放在 `docs/ui/`。
- 开发任务完成后更新本文件中对应阶段状态。

## 8. 工作项统计

| 阶段 | S | M | Smoke 测试 | 异常测试 | 集成门控 | 总计 |
|------|---|---|------------|----------|----------|------|
| 阶段 0 | 3 | 0 | 0 | 0 | 1 | 4 |
| 阶段 1 | 8 | 4 | 3 | 3 | 4 | 16 |
| 阶段 2 | 8 | 4 | 4 | 4 | 4 | 16 |
| 阶段 3 | 4 | 2 | 2 | 2 | 2 | 8 |
| 阶段 4 | 7 | 3 | 3 | 3 | 3 | 13 |
| 阶段 5 | 6 | 3 | 3 | 3 | 3 | 12 |
| 阶段 6 | 6 | 3 | 3 | 3 | 3 | 12 |
| 阶段 7 | 4 | 3 | 1 | 3 | 3 | 10 |
| **合计** | **46** | **22** | **19** | **21** | **23** | **91** |

## 9. 审核记录

| 日期 | 审核人 | 评分 | 结果 | 备注 |
|------|--------|------|------|------|
| 2026-07-04 | AI Assistant | 92/100 | 通过 | 已补齐局部门控、测试/门控 Notes、发布性能测试命名和并行开发风险。 |
