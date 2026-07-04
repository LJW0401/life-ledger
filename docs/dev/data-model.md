# life-ledger 数据模型设计

> 创建日期：2026-07-04
> 状态：已完成
> 版本：v1.0
> 关联文档：`docs/dev/prd.md`、`docs/dev/sad.md`

本文定义首版 SQLite schema、字段约束、索引和 migration 顺序。SQLite 是唯一运行时数据源，Excel 只作为账单导入导出的交换格式。

## 1. 设计原则

- 所有表使用 `TEXT` 主键保存应用生成的 UUID。
- 时间戳统一使用 UTC ISO-8601 字符串，页面按配置时区展示。
- 金额使用整数分保存，避免浮点误差。
- 删除首版采用物理删除；删除账单、重要日期或决策时写入 `audit_events`。
- 批量导入必须在单个 transaction 内完成，任一行失败不能产生部分写入。
- migration 由 Go `embed` 内嵌 SQL 文件执行，不使用外部 migration 工具。

## 2. Migration 顺序

| 版本 | 文件 | 内容 |
|------|------|------|
| 001 | `001_core.sql` | `schema_migrations`、业务表、标签和审计基础表。 |
| 002 | `002_auth.sql` | 设备会话、登录失败限速、CSRF 相关字段。 |
| 003 | `003_indexes.sql` | 查询、筛选、安全和性能索引。 |

`schema_migrations` 记录已经成功执行的版本。migration 在 transaction 中执行；失败时服务启动失败，并输出失败版本号。

## 3. 表结构

### 3.1 `schema_migrations`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `version` | INTEGER | PRIMARY KEY | migration 版本。 |
| `name` | TEXT | NOT NULL | migration 文件名。 |
| `applied_at` | TEXT | NOT NULL | 执行时间。 |

### 3.2 `important_dates`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 日期记录 ID。 |
| `title` | TEXT | NOT NULL | 标题。 |
| `date` | TEXT | NOT NULL | 日期，格式 `YYYY-MM-DD`。 |
| `date_type` | TEXT | NOT NULL | 类型，例如生日、纪念日、证件、缴费。 |
| `repeat_rule` | TEXT | NOT NULL DEFAULT '不重复' | `不重复`、`每年`、`每月`、`每周`。 |
| `note` | TEXT | NOT NULL DEFAULT '' | 备注。 |
| `created_at` | TEXT | NOT NULL | 创建时间。 |
| `updated_at` | TEXT | NOT NULL | 更新时间。 |

约束：

- `title`、`date`、`date_type` 不能为空。
- `repeat_rule` 只能是首版枚举值。

### 3.3 `transactions`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 账单 ID。 |
| `occurred_date` | TEXT | NOT NULL | 日期，格式 `YYYY-MM-DD`。 |
| `occurred_time` | TEXT | NOT NULL | 时间，格式 `HH:MM`。 |
| `type` | TEXT | NOT NULL | `收入` 或 `支出`。 |
| `amount_cents` | INTEGER | NOT NULL | 金额，单位分，必须大于 0。 |
| `category` | TEXT | NOT NULL | 分类。 |
| `include_income` | INTEGER | NOT NULL | 是否计入收支，0/1。 |
| `include_budget` | INTEGER | NOT NULL | 是否计入预算，0/1，仅支出生效。 |
| `ledger` | TEXT | NOT NULL | 所属账本。 |
| `counterparty` | TEXT | NOT NULL DEFAULT '' | 对象。 |
| `account` | TEXT | NOT NULL DEFAULT '' | 账户。 |
| `note` | TEXT | NOT NULL DEFAULT '' | 备注。 |
| `created_at` | TEXT | NOT NULL | 创建时间。 |
| `updated_at` | TEXT | NOT NULL | 更新时间。 |

约束：

- `amount_cents > 0`。
- `type IN ('收入', '支出')`。
- `include_income`、`include_budget` 只能为 0 或 1。
- 收入账单保存 `include_budget=1` 时，预算统计仍必须忽略该记录。

### 3.4 `budgets`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 预算 ID。 |
| `month` | TEXT | NOT NULL | 月份，格式 `YYYY-MM`。 |
| `category` | TEXT | NOT NULL | 分类。 |
| `amount_cents` | INTEGER | NOT NULL | 预算金额，单位分。 |
| `created_at` | TEXT | NOT NULL | 创建时间。 |
| `updated_at` | TEXT | NOT NULL | 更新时间。 |

约束：

- `amount_cents > 0`。
- `UNIQUE(month, category)`。

### 3.5 `decisions`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 决策 ID。 |
| `title` | TEXT | NOT NULL | 标题。 |
| `background` | TEXT | NOT NULL DEFAULT '' | 背景。 |
| `final_choice` | TEXT | NOT NULL DEFAULT '' | 最终选择。 |
| `status` | TEXT | NOT NULL | `进行中`、`待复盘`、`已归档`。 |
| `review_date` | TEXT | NULL | 复盘日期，格式 `YYYY-MM-DD`。 |
| `review_note` | TEXT | NOT NULL DEFAULT '' | 复盘内容。 |
| `created_at` | TEXT | NOT NULL | 创建时间。 |
| `updated_at` | TEXT | NOT NULL | 更新时间。 |

状态规则：

- 未做最终选择时通常保存为 `进行中`。
- `review_date <= today` 且未归档的记录查询时归入 `待复盘`。
- 完成复盘后保存为 `已归档`。

### 3.6 `decision_options`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 候选方案 ID。 |
| `decision_id` | TEXT | NOT NULL | 所属决策。 |
| `name` | TEXT | NOT NULL | 方案名称。 |
| `pros` | TEXT | NOT NULL DEFAULT '' | 优点。 |
| `cons` | TEXT | NOT NULL DEFAULT '' | 缺点。 |
| `note` | TEXT | NOT NULL DEFAULT '' | 备注。 |
| `sort_order` | INTEGER | NOT NULL | 展示顺序。 |

外键：

- `decision_id REFERENCES decisions(id) ON DELETE CASCADE`。

### 3.7 `tags`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 标签 ID。 |
| `name` | TEXT | NOT NULL UNIQUE | 标签名。 |
| `created_at` | TEXT | NOT NULL | 创建时间。 |

标签名去掉首尾空格后不能为空。首版建议长度上限为 32 个字符。

### 3.8 `entity_tags`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `entity_type` | TEXT | NOT NULL | `important_date`、`transaction`、`decision`。 |
| `entity_id` | TEXT | NOT NULL | 业务对象 ID。 |
| `tag_id` | TEXT | NOT NULL | 标签 ID。 |
| `created_at` | TEXT | NOT NULL | 关联时间。 |

主键：

- `PRIMARY KEY(entity_type, entity_id, tag_id)`。

说明：SQLite 无法对多态关联直接建外键，业务删除对象时必须显式清理关联。

### 3.9 `device_sessions`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 设备会话 ID。 |
| `device_name` | TEXT | NOT NULL | 设备名称。 |
| `token_hash` | TEXT | NOT NULL UNIQUE | 会话 token 哈希。 |
| `csrf_token_hash` | TEXT | NOT NULL | CSRF token 哈希。 |
| `user_agent` | TEXT | NOT NULL DEFAULT '' | User-Agent。 |
| `first_login_at` | TEXT | NOT NULL | 首次登录时间。 |
| `last_seen_at` | TEXT | NOT NULL | 最后访问时间。 |
| `last_seen_ip` | TEXT | NOT NULL DEFAULT '' | 最后访问 IP。 |
| `expires_at` | TEXT | NOT NULL | 固定过期时间。 |
| `revoked_at` | TEXT | NULL | 撤销时间。 |

规则：

- Cookie 保存原始随机 token，数据库只保存哈希。
- 有效期固定 7 天，访问不会延长 `expires_at`。
- 退出登录或撤销设备后设置 `revoked_at`。

### 3.10 `login_failures`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 记录 ID。 |
| `username` | TEXT | NOT NULL | 尝试登录用户名。 |
| `client_ip` | TEXT | NOT NULL | 解析后的客户端 IP。 |
| `failure_count` | INTEGER | NOT NULL | 窗口内失败次数。 |
| `window_started_at` | TEXT | NOT NULL | 计数窗口开始时间。 |
| `last_failed_at` | TEXT | NOT NULL | 最近失败时间。 |
| `locked_until` | TEXT | NULL | 锁定截止时间。 |

规则：

- 默认 10 分钟内失败 5 次锁定 15 分钟。
- 登录成功后清理同用户名和同 IP 的活跃失败记录。

### 3.11 `audit_events`

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | TEXT | PRIMARY KEY | 审计事件 ID。 |
| `event_type` | TEXT | NOT NULL | 事件类型。 |
| `occurred_at` | TEXT | NOT NULL | 发生时间。 |
| `client_ip` | TEXT | NOT NULL DEFAULT '' | 客户端 IP。 |
| `device_id` | TEXT | NULL | 设备 ID。 |
| `user_agent` | TEXT | NOT NULL DEFAULT '' | User-Agent。 |
| `resource_type` | TEXT | NOT NULL DEFAULT '' | 资源类型。 |
| `resource_id` | TEXT | NOT NULL DEFAULT '' | 资源 ID。 |
| `result` | TEXT | NOT NULL | `success` 或 `failure`。 |
| `reason` | TEXT | NOT NULL DEFAULT '' | 脱敏失败原因。 |
| `metadata_json` | TEXT | NOT NULL DEFAULT '{}' | 脱敏元数据。 |

审计不得记录明文密码、`password_hash`、`session_secret`、session token、Cookie、CSRF token、Excel 原始内容或完整业务记录内容。

## 4. 索引

| 索引 | 表 | 字段 | 用途 |
|------|----|------|------|
| `idx_transactions_date_type` | `transactions` | `occurred_date, type` | 账单列表和月份筛选。 |
| `idx_transactions_category_date` | `transactions` | `category, occurred_date` | 分类统计和预算消耗。 |
| `idx_transactions_include_income_date` | `transactions` | `include_income, occurred_date` | 收入、支出、余额统计。 |
| `idx_transactions_budget` | `transactions` | `include_budget, category, occurred_date` | 预算消耗统计。 |
| `idx_important_dates_date_repeat` | `important_dates` | `date, repeat_rule` | 日期页排序和重复规则筛选。 |
| `idx_decisions_status_review` | `decisions` | `status, review_date` | 决策分组和待复盘查询。 |
| `idx_entity_tags_entity` | `entity_tags` | `entity_type, entity_id` | 业务对象查标签。 |
| `idx_entity_tags_tag` | `entity_tags` | `tag_id, entity_type` | 按标签筛选。 |
| `idx_device_sessions_token_hash` | `device_sessions` | `token_hash` | 会话校验。 |
| `idx_device_sessions_state` | `device_sessions` | `expires_at, revoked_at` | 设备管理列表。 |
| `idx_login_failures_identity` | `login_failures` | `username, client_ip` | 登录失败限速。 |
| `idx_audit_events_time_type` | `audit_events` | `occurred_at, event_type` | 审计查询。 |

## 5. 备份元数据

备份包中的 `backup-meta.json` 不入库，生成时读取当前 schema 版本。结构：

```json
{
  "app_version": "0.1.0",
  "backup_time": "2026-07-04T12:00:00Z",
  "schema_version": 3,
  "files": ["life-ledger.db", "config.toml"]
}
```
