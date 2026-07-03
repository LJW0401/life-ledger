# life-ledger

个人生活管理服务，记录重要日期、收支流水和关键决策，并支持提醒、分析与复盘。

## 技术栈

- 后端：Go
- 程序入口：`cmd/server/main.go`
- HTTP 路由：`chi`，或先使用标准库 `net/http`
- 前端：React + TypeScript + Vite
- UI：Tailwind CSS + shadcn/ui
- 数据库：SQLite
- 数据访问：手写 SQL 起步
- 数据库迁移：应用启动时执行内嵌 SQL migration
- Excel 导入导出：`github.com/xuri/excelize/v2`
- 配置文件：外部 `config.toml`
- 配置解析：`github.com/pelletier/go-toml/v2`
- 静态资源打包：Go `embed` 内嵌前端 `dist`
- 定时任务：`robfig/cron`，或自写轻量 scheduler
- 部署方式：单个二进制文件 + `config.toml` + 数据目录

## 部署形态

构建后部署目录建议如下：

```text
life-ledger/
  life-ledger          可执行文件
  config.toml          服务配置
  data/
    life-ledger.db     SQLite 数据库
```

默认运行方式：

```bash
./life-ledger
```

程序默认读取当前目录下的 `config.toml`。命令行参数只保留特殊场景使用的 `--config`：

```bash
./life-ledger --config /path/to/config.toml
```

## 源码结构

```text
life-ledger/
  docs/ui/             UI 设计文档、页面草图和交互说明
  cmd/server/          程序入口，最终编译成 life-ledger 二进制
  internal/config/     读取和校验 config.toml
  internal/api/        HTTP API
  internal/domain/     账单、重要日期、决策等业务逻辑
  internal/db/         SQLite、SQL、内嵌 migration
  internal/jobs/       提醒、复盘、定时任务
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

[reminder]
enabled = true
check_interval_seconds = 60

[export]
timezone = "Asia/Shanghai"
```

## 数据存储

账单、重要日期、决策信息都存储在 SQLite 数据库中。数据库是唯一数据源，Excel 只作为账单导入导出的交换格式。

核心数据表建议按职责拆分：

- `transactions`：账单、收入、支出流水
- `important_dates`：生日、纪念日、证件到期、缴费日等重要日期
- `decisions`：关键决策、决策背景、选项、结果和复盘信息
- `reminders`：提醒任务，可关联账单、日期或决策
- `tags` / `entity_tags`：跨业务对象复用的标签

## Excel 账单导入导出

账单 Excel 固定使用 `.xlsx` 格式。

建议 API：

```text
GET  /api/bills/export.xlsx       导出账单 Excel
GET  /api/bills/template.xlsx     下载账单导入模板
POST /api/bills/import.xlsx       上传账单 Excel 并导入
```

导入规则：

- 先校验整张表。
- 有错误时返回具体行号和原因，不写入数据库。
- 校验通过后，在一个 SQLite transaction 中一次性写入。
- 导出时按数据库当前内容生成 Excel。
