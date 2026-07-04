# life-ledger

个人生活管理服务，记录重要日期、收支流水和关键决策，并支持分析与复盘。

## 技术栈

- 后端：Go
- 程序入口：`cmd/server/main.go`
- HTTP 路由：标准库 `net/http`
- 前端：React + TypeScript + Vite
- UI：普通 CSS + `lucide-react` 图标
- 数据库：SQLite
- 数据访问：手写 SQL
- 数据库迁移：应用启动时执行内嵌 SQL migration
- Excel 导入导出：`github.com/xuri/excelize/v2`
- 配置文件：外部 `config.toml`
- 配置解析：`github.com/pelletier/go-toml/v2`
- 静态资源打包：Go `embed` 内嵌前端 `dist`
- 部署方式：单个二进制文件 + `config.toml` + 数据目录

## 部署形态

构建后部署目录建议如下：

```text
life-ledger/
  life-ledger          可执行文件
  config.toml          服务配置
  data/
    life-ledger.db     SQLite 数据库
  backups/             本地备份
```

默认运行方式：

```bash
./life-ledger
```

程序默认读取当前目录下的 `config.toml`。命令行参数只保留特殊场景使用的 `--config`：

```bash
./life-ledger --config /path/to/config.toml
```

首次部署前生成安全配置值：

```bash
./life-ledger hash-password
./life-ledger generate-secret
```

备份：

```bash
./life-ledger backup
```

Caddy 负责公网 HTTPS 和反向代理，应用默认只监听 `127.0.0.1:8080`：

```caddyfile
life.example.com {
  reverse_proxy 127.0.0.1:8080
}
```

## 本地开发

首次准备依赖：

```bash
go mod tidy
npm --prefix web install
npm --prefix web exec playwright install chromium
```

生成部署二进制：

```bash
make build
```

产物位置：

```text
bin/life-ledger
```

常用门控：

```bash
test -z "$(gofmt -l ./cmd ./internal ./web/*.go)"
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/server
npm run typecheck
npm run lint
npm run build
npm run test:e2e
```

E2E 测试会自动构建前端，并用临时 `config.toml` 和临时 SQLite 数据目录启动本地服务。

## 源码结构

```text
life-ledger/
  docs/dev/            PRD、SAD、TPD、API、数据模型、部署和任务计划
  docs/ui/             UI 设计文档、页面草图和交互说明
  cmd/server/          程序入口，最终编译成 life-ledger 二进制
  internal/config/     读取和校验 config.toml
  internal/api/        HTTP API
  internal/domain/     账单、重要日期、决策等业务逻辑
  internal/db/         SQLite、SQL、内嵌 migration
  web/                 React 前端
```

## 配置文件示例

```toml
[server]
host = "127.0.0.1"
port = 8080

[data]
dir = "./data"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = ""
session_secret = ""
session_days = 7

[security]
trusted_proxies = ["127.0.0.1"]
login_failure_window_minutes = 10
login_failure_limit = 5
login_lock_minutes = 15
cookie_secure = true

[export]
timezone = "Asia/Shanghai"
max_upload_mb = 5
max_import_rows = 5000

[backup]
dir = "./backups"
```

`password_hash` 和 `session_secret` 必须由辅助命令生成后填入。`config.toml` 权限必须是 `600`，数据目录权限必须是 `700`。

## 数据存储

账单、重要日期、决策信息都存储在 SQLite 数据库中。数据库是唯一数据源，Excel 只作为账单导入导出的交换格式。

核心数据表建议按职责拆分：

- `transactions`：账单、收入、支出流水
- `important_dates`：生日、纪念日、证件到期、缴费日等重要日期
- `decisions`：关键决策、决策背景、选项、结果和复盘信息
- `tags` / `entity_tags`：跨业务对象复用的标签

## Excel 账单导入导出

账单 Excel 固定使用 `.xlsx` 格式。

建议 API：

```text
GET  /api/transactions/export.xlsx       导出账单 Excel
GET  /api/transactions/template.xlsx     下载账单导入模板
POST /api/transactions/import.xlsx       上传账单 Excel 并导入
```

导入规则：

- 固定列：日期、时间、类型、金额、分类、计入收支、计入预算、所属账本、对象、账户、标签、备注。
- 必填列：日期、时间、类型、金额、分类、计入收支、计入预算、所属账本。
- 先校验整张表。
- 有错误时返回具体行号和原因，不写入数据库。
- 校验通过后，在一个 SQLite transaction 中一次性写入。
- 导出时按数据库当前内容生成 Excel。
