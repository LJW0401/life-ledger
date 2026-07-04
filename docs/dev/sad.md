# life-ledger 软件架构文档

> 创建日期：2026-07-04
> 状态：待审核
> 版本：v0.1
> 模式：完整
> 关联需求文档：`docs/dev/prd.md`
> 关联需求状态：草案，需在 PRD 复审通过后同步复核本架构文档

## 1. 架构概述

### 1.1 架构风格

life-ledger 采用单体 Web 应用架构：

- 后端使用 Go 提供 HTTP API、静态资源服务、配置读取、认证授权、SQLite 访问、Excel 导入导出和备份命令。
- 前端使用 React 单页应用，构建产物通过 Go `embed` 内嵌进最终二进制。
- 数据库使用本地 SQLite 文件，作为唯一运行时数据源。
- 线上部署由 Caddy 负责 HTTPS 和反向代理，应用默认只监听 `127.0.0.1`。

选择单体架构的原因：

- 首版是个人自用，不需要多租户、分布式扩容或多服务协作。
- 单二进制部署是硬约束，单体架构最符合部署目标。
- SQLite、Excel、会话和业务数据都在同一进程内处理，可以减少外部依赖和运维复杂度。
- 安全边界集中在一个进程内，便于统一做认证、CSRF、审计和日志脱敏。

### 1.2 架构图

HTML 可视化版本见 `docs/dev/sad-diagram.html`。

```text
Browser
  |
  | HTTPS
  v
Caddy
  |
  | HTTP reverse_proxy 127.0.0.1:8080
  v
life-ledger binary
  |
  |-- embedded React dist
  |-- HTTP API
  |-- auth/session/csrf middleware
  |-- business services
  |-- Excel import/export
  |-- backup command
  |
  v
SQLite database + config.toml + backup files
```

### 1.3 设计原则

- **本地优先**：SQLite 是唯一数据源，Excel 只作为导入导出格式。
- **失败显式**：配置错误、migration 失败、权限过宽、导入校验失败都应明确失败，不静默降级。
- **单一职责**：HTTP、业务逻辑、数据访问、Excel、认证、安全审计分别拆分模块。
- **最小外部依赖**：不引入 Redis、消息队列、外部迁移工具或独立前端部署服务。
- **默认安全**：默认监听本机地址，敏感配置强校验，写操作强制 CSRF 校验。
- **可迁移**：备份包应包含恢复所需的 SQLite、配置和元数据。

## 2. 技术栈

| 类别 | 选择 | 版本 | 说明 |
|------|------|------|------|
| 后端语言 | Go | 由 `go.mod` 锁定 | 生成单个 Linux 二进制，负责 HTTP、CLI、SQLite 和静态资源内嵌。 |
| HTTP 路由 | `chi` | 由 `go.mod` 锁定 | 路由和 middleware 组合清晰，复杂度低于完整 Web 框架。 |
| 数据库 | SQLite | 系统随驱动提供 | 单文件数据库，满足个人数据规模和备份迁移需求。 |
| SQLite 驱动 | `modernc.org/sqlite` | 由 `go.mod` 锁定 | 优先保持纯 Go 构建，降低 CGO 和交叉编译复杂度。 |
| 数据访问 | 手写 SQL | N/A | 首版表结构可控，不引入 ORM。 |
| 数据迁移 | 内嵌 SQL migration | N/A | 使用 Go `embed` 加载 migration，不使用 goose。 |
| 配置解析 | `github.com/pelletier/go-toml/v2` | 由 `go.mod` 锁定 | 读取 `config.toml`。 |
| 密码哈希 | `golang.org/x/crypto/bcrypt` | 由 `go.mod` 锁定 | 默认 cost=12。 |
| Excel | `github.com/xuri/excelize/v2` | 由 `go.mod` 锁定 | 账单 `.xlsx` 模板、导入、导出。 |
| 前端 | React + TypeScript + Vite | 由 `package.json` 锁定 | 三个一级页面的 SPA。 |
| UI | Tailwind CSS + shadcn/ui | 由 `package.json` 锁定 | 适合表单、表格、弹窗和管理类界面。 |
| 反向代理 | Caddy | 系统安装版本 | 负责 HTTPS、域名和反向代理。 |
| 定时任务 | 暂不引入 | N/A | PRD 已移除提醒，首版不需要后台定时提醒。 |

## 3. 模块设计

### 3.1 模块总览

```text
cmd/server
  -> internal/app
       -> internal/config
       -> internal/db
       -> internal/api
       -> internal/web
       -> internal/backup

internal/api
  -> internal/auth
  -> internal/security
  -> internal/domain/*
  -> internal/excel
  -> internal/audit

internal/domain/*
  -> internal/db
  -> internal/audit
```

### 3.2 程序入口模块

- **职责**：解析 CLI 子命令，加载配置，初始化依赖，启动 HTTP 服务或执行一次性命令。
- **对外接口**：`./life-ledger`、`./life-ledger --config`、`hash-password`、`generate-secret`、`backup`。
- **依赖**：配置模块、应用组装模块、备份模块、安全辅助模块。
- **关键设计决策**：入口只做编排，不写业务规则和 SQL。

### 3.3 配置模块

- **职责**：读取 `config.toml`、应用默认值、校验必填项和敏感配置。
- **对外接口**：`Load(path) Config`、`Validate(config) error`。
- **依赖**：TOML 解析库、文件权限检查。
- **关键设计决策**：配置错误直接阻止启动；不支持明文密码配置。

### 3.4 数据库模块

- **职责**：打开 SQLite、设置连接参数、执行 migration、提供 transaction 边界和 repository 基础能力。
- **对外接口**：`Open(config) DB`、`Migrate(db) error`、`WithinTx(ctx, fn)`。
- **依赖**：SQLite 驱动、内嵌 SQL migration。
- **关键设计决策**：migration 失败必须阻止服务启动；所有批量导入在 transaction 中完成。

### 3.5 HTTP API 模块

- **职责**：注册路由、绑定 middleware、处理请求响应、统一错误格式。
- **对外接口**：REST 风格 HTTP API。
- **依赖**：认证、安全、业务服务、Excel、审计。
- **关键设计决策**：API 层不直接操作 SQL；写操作统一经过认证和 CSRF middleware。

### 3.6 认证与会话模块

- **职责**：登录、退出、设备会话、Cookie、会话 token 哈希、设备撤销、登录失败限速。
- **对外接口**：登录 API、退出 API、设备管理 API、认证 middleware。
- **依赖**：配置模块、数据库模块、审计模块、密码哈希库、密码学随机数。
- **关键设计决策**：Cookie 保存原始随机 token 和签名，数据库只保存 token 哈希；7 天固定过期，不做滑动续期。

### 3.7 安全模块

- **职责**：CSRF 校验、安全响应头、CORS 策略、可信反代 IP 解析、日志脱敏工具。
- **对外接口**：HTTP middleware 和通用安全工具函数。
- **依赖**：认证模块、配置模块。
- **关键设计决策**：首版不开放跨源 API；只在请求来自可信反代地址时读取 `X-Forwarded-For`。

### 3.8 业务领域模块

- **职责**：实现重要日期、账单、预算、决策、标签的业务规则。
- **对外接口**：领域 service 和 repository。
- **依赖**：数据库模块、审计模块。
- **关键设计决策**：领域模块只表达业务语义，HTTP 入参和 SQLite 行结构不泄漏到页面之外的层级。

业务子模块：

- `importantdates`：重要日期 CRUD、重复规则校验。
- `transactions`：账单 CRUD、筛选、收支统计基础。
- `budgets`：月份 + 分类预算、预算消耗计算。
- `decisions`：决策 CRUD、候选方案、状态和复盘。
- `tags`：标签创建、关联、筛选。

### 3.9 Excel 模块

- **职责**：生成账单模板、解析导入文件、校验整表、导出 `.xlsx`。
- **对外接口**：模板生成、导入预校验、导出生成。
- **依赖**：业务领域模块、Excel 库。
- **关键设计决策**：导入先校验整表，再由业务服务在一个 SQLite transaction 中写入；错误只返回行号、列名和原因。

### 3.10 审计模块

- **职责**：写入安全事件和高风险操作事件，提供脱敏摘要。
- **对外接口**：`Record(event)`、`List(filter)`。
- **依赖**：数据库模块。
- **关键设计决策**：SQLite `audit_events` 是审计事实来源，应用日志只输出脱敏摘要。

### 3.11 静态资源模块

- **职责**：内嵌并服务前端 `dist`，支持 SPA 路由 fallback。
- **对外接口**：静态文件 handler。
- **依赖**：Go `embed`。
- **关键设计决策**：非 `/api/*` 路由回退到 `index.html`，以支持刷新 `/important-dates`、`/transactions`、`/decisions`。

### 3.12 备份模块

- **职责**：执行 `backup` 命令，生成包含 SQLite、`config.toml` 和 `backup-meta.json` 的本地备份包。
- **对外接口**：`./life-ledger backup`。
- **依赖**：配置模块、数据库文件、文件系统。
- **关键设计决策**：备份命令不启动 HTTP 服务，不修改数据库；失败时非零退出。

## 4. 数据架构

### 4.1 数据模型

核心实体：

- `schema_migrations`：记录已执行 migration。
- `important_dates`：重要日期记录。
- `transactions`：账单流水。
- `budgets`：月份 + 分类预算。
- `decisions`：决策主体。
- `decision_options`：决策候选方案。
- `tags`：标签字典。
- `entity_tags`：标签和业务对象的多态关联。
- `device_sessions`：登录设备和会话。
- `login_failures`：登录失败限速状态。
- `audit_events`：结构化审计事件。

关系概览：

```text
transactions
  -> budgets by month + category when type=支出 and include_budget=是
  -> tags through entity_tags

important_dates
  -> tags through entity_tags

decisions
  -> decision_options
  -> tags through entity_tags

device_sessions
  -> audit_events by device_id
```

### 4.2 数据存储

- SQLite 数据库文件位于配置指定的数据目录下。
- `config.toml` 保存服务端口、数据目录、用户名、密码哈希、`session_secret` 和安全默认值。
- 上传的 Excel 文件不长期落盘；解析完成后释放临时资源。
- 备份包输出到配置或命令默认的备份目录，包含数据库、配置和元数据。

### 4.3 数据流

账单 Excel 导入：

```text
upload .xlsx
  -> API size/type check
  -> Excel parse first sheet
  -> header and row validation
  -> transaction begin
  -> insert transactions
  -> audit event
  -> transaction commit
```

登录和设备会话：

```text
login request
  -> login rate check
  -> bcrypt verify
  -> generate session token and csrf token
  -> store token hashes in SQLite
  -> set HttpOnly cookie
  -> audit event
```

写操作 API：

```text
request
  -> trusted proxy IP parsing
  -> session cookie verification
  -> csrf token verification
  -> domain validation
  -> repository write
  -> audit event when required
  -> JSON response
```

## 5. 接口设计

### 5.1 内部接口

- API 层调用领域 service，不直接操作 repository。
- 领域 service 通过 repository 读写 SQLite。
- 多表写入和 Excel 导入通过数据库模块提供的 transaction 边界执行。
- 审计模块由认证、Excel 和高风险业务操作显式调用。

### 5.2 外部接口

外部接口采用同源 REST + JSON：

- 页面和 API 同源部署。
- `/api/*` 返回 JSON 或 `.xlsx` 文件。
- 非 `/api/*` 路径由前端 SPA 处理。
- 写操作使用 `POST`、`PUT`、`PATCH`、`DELETE`，必须携带 `X-CSRF-Token`。
- 错误响应使用统一 JSON 结构。

错误响应建议：

```json
{
  "error": {
    "code": "validation_failed",
    "message": "请求参数不合法",
    "details": []
  }
}
```

### 5.3 关键接口定义

认证和会话：

```text
POST   /api/auth/login
POST   /api/auth/logout
GET    /api/session
GET    /api/devices
DELETE /api/devices/{id}
```

重要日期：

```text
GET    /api/important-dates
POST   /api/important-dates
GET    /api/important-dates/{id}
PUT    /api/important-dates/{id}
DELETE /api/important-dates/{id}
```

账单和预算：

```text
GET    /api/transactions
POST   /api/transactions
GET    /api/transactions/{id}
PUT    /api/transactions/{id}
DELETE /api/transactions/{id}

GET    /api/transactions/template.xlsx
POST   /api/transactions/import.xlsx
GET    /api/transactions/export.xlsx

GET    /api/budgets
POST   /api/budgets
PUT    /api/budgets/{id}
DELETE /api/budgets/{id}
```

决策：

```text
GET    /api/decisions
POST   /api/decisions
GET    /api/decisions/{id}
PUT    /api/decisions/{id}
DELETE /api/decisions/{id}
```

## 6. 项目结构

```text
life-ledger/
  cmd/server/
    main.go
  internal/
    app/
    config/
    db/
      migrations/
    api/
    auth/
    security/
    audit/
    domain/
      importantdates/
      transactions/
      budgets/
      decisions/
      tags/
    excel/
    backup/
    web/
  web/
    src/
    package.json
    vite.config.ts
  docs/
    dev/
      prd.md
      sad.md
      data-model.md
      api-design.md
      roadmap.md
      phase-01-mvp.md
    ui/
  config.example.toml
  go.mod
  package.json
```

目录职责：

- `cmd/server/`：二进制入口和 CLI 子命令分发。
- `internal/app/`：应用依赖组装和生命周期管理。
- `internal/config/`：配置读取、默认值和校验。
- `internal/db/`：SQLite 打开、migration、transaction 和 repository 基础设施。
- `internal/api/`：HTTP 路由、handler、请求响应模型。
- `internal/auth/`：登录、退出、设备会话和登录失败限速。
- `internal/security/`：CSRF、安全响应头、CORS、可信反代和脱敏。
- `internal/audit/`：审计事件写入和查询。
- `internal/domain/`：业务规则和领域服务。
- `internal/excel/`：账单 Excel 模板、导入、导出。
- `internal/backup/`：备份命令。
- `internal/web/`：前端构建产物 embed 和 SPA fallback。
- `web/`：React 前端源码。

## 7. 部署架构

### 7.1 运行环境

- Linux 云服务器。
- Caddy 作为公网入口。
- 应用进程监听 `127.0.0.1:<port>`。
- 数据目录权限 `700`，`config.toml` 和 SQLite 数据库文件权限 `600`。

### 7.2 部署方式

部署目录：

```text
life-ledger/
  life-ledger
  config.toml
  data/
    life-ledger.db
  backups/
```

启动方式：

```bash
./life-ledger
```

Caddy 示例：

```caddyfile
life.example.com {
  reverse_proxy 127.0.0.1:8080
}
```

应用不依赖 Caddy Basic Auth。Caddy 只负责 HTTPS、域名和反向代理，应用负责登录态和 API 访问控制。

### 7.3 CI/CD

首版以本地构建和手动部署为主：

```text
npm install
npm run build
go test ./...
go build -o dist/life-ledger ./cmd/server
scp dist/life-ledger server:/opt/life-ledger/
```

后续可以补充 GitHub Actions，但不是首版架构依赖。

## 8. 安全门控命令

后端：

- 格式检查：`test -z "$(gofmt -l ./cmd ./internal)"`
- 静态检查：`go vet ./...`
- 单元测试：`go test ./...`
- 竞态测试：`go test -race ./...`
- 构建检查：`go build ./cmd/server`

前端：

- 类型检查：`npm run typecheck`
- 代码检查：`npm run lint`
- 构建检查：`npm run build`

集成和安全冒烟：

- 后端集成测试：`go test ./internal/...`
- E2E 冒烟测试：`npm run test:e2e`
- 发布前完整门控：`go vet ./... && go test ./... && npm run typecheck && npm run lint && npm run build`

## 9. 风险与应对

| 风险 | 影响 | 概率 | 应对措施 |
|------|------|------|----------|
| SQLite 文件损坏或误删 | 丢失个人核心数据 | 中 | 提供手动备份命令；部署文档明确恢复流程；写操作使用 transaction。 |
| 云服务器公网暴露应用端口 | 绕过 Caddy 的 HTTPS 入口 | 中 | 默认监听 `127.0.0.1`；配置公网地址需显式修改；部署文档要求防火墙限制。 |
| 会话 Cookie 或 token 泄露 | 未授权访问个人数据 | 中 | HttpOnly、Secure、SameSite、token 哈希存储、设备撤销、7 天固定过期。 |
| Excel 导入数据污染 | 错误账单批量写入 | 中 | 先校验整表再写入；错误时不部分写入；返回精确行列原因。 |
| 业务模块直接耦合 SQL | 后续修改字段困难 | 中 | 领域 service 和 repository 分层；SQL 集中在 repository。 |
| 前后端接口漂移 | 页面可用性下降 | 中 | 后续补 `api-design.md`，接口变更先改文档和测试。 |
| 备份包包含登录能力 | 备份泄露等同账号和数据泄露 | 中 | 备份文档明确风险；默认不上传云端；建议用户自行加密保存。 |
| 纯 Go SQLite 驱动兼容性问题 | 构建或运行时 SQL 行为差异 | 低 | 限制 SQL 方言；用集成测试覆盖 migration 和核心 CRUD；必要时切换到 `mattn/go-sqlite3`。 |

## 10. 架构决策记录

### ADR-1: 采用单体 Web 应用

- **背景**：项目面向个人自用，要求单二进制部署。
- **方案对比**：
  - 单体 Web 应用：部署简单，依赖少，符合当前规模。
  - 前后端分离独立部署：灵活，但增加部署和反代复杂度。
  - 微服务：明显过度设计。
- **决策**：采用 Go 单体 Web 应用，内嵌 React 构建产物。
- **影响**：所有核心能力在一个进程内交付，后续扩展需要保持模块边界清晰。

### ADR-2: 使用 SQLite 作为唯一数据源

- **背景**：账单、重要日期、决策、设备和审计都需要持久化。
- **方案对比**：
  - SQLite：单文件、易备份、运维成本低。
  - PostgreSQL：能力强，但个人自用场景运维成本偏高。
  - Excel 文件：适合交换，不适合作为运行时数据源。
- **决策**：SQLite 作为唯一运行时数据源，Excel 只做导入导出。
- **影响**：需要做好 migration、transaction 和备份流程。

### ADR-3: 不使用外部 migration 工具

- **背景**：用户明确不需要 goose，并希望单二进制部署。
- **方案对比**：
  - 内嵌 SQL migration：部署简单，与二进制版本绑定。
  - goose：成熟，但增加额外工具和流程。
- **决策**：使用 Go `embed` 内嵌 migration，并用 `schema_migrations` 表记录执行状态。
- **影响**：需要自行实现 migration 执行顺序、幂等检查和失败处理。

### ADR-4: 应用负责登录，Caddy 只负责 HTTPS

- **背景**：需要设备记录、7 天免登录、设备撤销和审计。
- **方案对比**：
  - Caddy Basic Auth：简单，但无法满足设备管理和审计。
  - 应用内单用户登录：实现成本更高，但满足产品需求。
- **决策**：Caddy 只负责 HTTPS 和反向代理，应用负责认证和会话。
- **影响**：必须实现 Cookie、CSRF、登录失败限速、设备撤销和审计。

### ADR-5: 使用 REST + JSON API

- **背景**：前端是 React SPA，后端是单体 Go API。
- **方案对比**：
  - REST + JSON：简单直观，适合 CRUD 和文件上传下载。
  - GraphQL：灵活但复杂度过高。
  - RPC：类型清晰但对浏览器表单和文件接口不如 REST 直观。
- **决策**：使用同源 REST + JSON，Excel 接口返回 `.xlsx` 文件。
- **影响**：需要后续在 `api-design.md` 中固定路由、请求体和错误码。

### ADR-6: 首版不引入后台提醒调度

- **背景**：PRD 已明确首版不包含提醒能力。
- **方案对比**：
  - 引入 cron/scheduler：为提醒预留能力，但当前没有使用场景。
  - 暂不引入：减少运行时复杂度。
- **决策**：首版不引入定时任务库。
- **影响**：重要日期只在页面查看时展示，不主动推送通知。
