# life-ledger API 设计

> 创建日期：2026-07-04
> 状态：已完成
> 版本：v1.0
> 关联文档：`docs/dev/prd.md`、`docs/dev/sad.md`、`docs/dev/data-model.md`

本文定义首版同源 REST API 契约。页面和 API 由同一个 Go 服务提供，`/api/*` 返回 JSON 或 `.xlsx` 文件，非 API 路径交给 React SPA。

## 1. 通用约定

### 1.1 认证和 CSRF

- 除 `POST /api/auth/login` 外，业务 API 默认需要登录态 Cookie。
- 登录态 Cookie 为 HttpOnly，SameSite=Lax；HTTPS 部署时设置 Secure。
- 登录成功响应和 `GET /api/session` 返回 `csrf_token`。
- `POST`、`PUT`、`PATCH`、`DELETE` 必须携带 `X-CSRF-Token`。
- `GET`、`HEAD`、`OPTIONS` 不要求 CSRF token。
- 退出登录或撤销设备后，对应 session token 和 CSRF token 失效。

### 1.2 错误格式

```json
{
  "error": {
    "code": "validation_failed",
    "message": "请求参数不合法",
    "details": [
      {
        "field": "amount",
        "reason": "金额必须大于 0"
      }
    ]
  }
}
```

常用错误码：

| HTTP | code | 场景 |
|------|------|------|
| 400 | `validation_failed` | 字段缺失、格式错误、非法枚举。 |
| 401 | `unauthorized` | 未登录、会话过期、会话撤销。 |
| 403 | `csrf_failed` | 写操作缺少或携带错误 CSRF token。 |
| 404 | `not_found` | 资源不存在。 |
| 409 | `conflict` | 唯一约束冲突或状态冲突。 |
| 413 | `payload_too_large` | Excel 上传超过 5MB。 |
| 500 | `internal_error` | 未预期服务端错误。 |

错误响应不得回显明文密码、Cookie、session token、CSRF token 或 Excel 原始整行内容。

### 1.3 分页和筛选

列表默认 `page=1&page_size=50`，最大 `page_size=200`。超过最大值时返回 400。

分页响应：

```json
{
  "items": [],
  "page": 1,
  "page_size": 50,
  "total": 0
}
```

### 1.4 安全响应头

所有页面和 API 响应设置：

- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: same-origin`

首版不开放跨源 API，不返回 `Access-Control-Allow-Origin: *`。

## 2. 认证和设备

### 2.1 登录

`POST /api/auth/login`

请求：

```json
{
  "username": "admin",
  "password": "plain-password",
  "device_name": "Chrome on Linux"
}
```

响应：

```json
{
  "session": {
    "expires_at": "2026-07-11T12:00:00Z"
  },
  "device": {
    "id": "dev_123",
    "device_name": "Chrome on Linux",
    "current": true
  },
  "csrf_token": "token"
}
```

行为：

- 正确凭据创建设备会话并设置 Cookie。
- 错误凭据写入登录失败记录和审计事件。
- 触发限速时不校验密码，直接返回 429。

### 2.2 退出

`POST /api/auth/logout`

响应：

```json
{
  "ok": true
}
```

行为：撤销当前设备会话，清除 Cookie，写入审计事件。

### 2.3 当前会话

`GET /api/session`

响应：

```json
{
  "authenticated": true,
  "device": {
    "id": "dev_123",
    "device_name": "Chrome on Linux",
    "last_seen_at": "2026-07-04T12:00:00Z",
    "expires_at": "2026-07-11T12:00:00Z"
  },
  "csrf_token": "token"
}
```

未登录时返回 401。

### 2.4 设备列表

`GET /api/devices`

响应：

```json
{
  "items": [
    {
      "id": "dev_123",
      "device_name": "Chrome on Linux",
      "user_agent": "Mozilla/5.0",
      "first_login_at": "2026-07-04T12:00:00Z",
      "last_seen_at": "2026-07-04T12:00:00Z",
      "last_seen_ip": "127.0.0.1",
      "expires_at": "2026-07-11T12:00:00Z",
      "revoked_at": null,
      "current": true
    }
  ]
}
```

### 2.5 撤销设备

`DELETE /api/devices/{id}`

响应：

```json
{
  "ok": true,
  "current_device_revoked": false
}
```

撤销当前设备时同时清除 Cookie，前端返回登录页。

## 3. 重要日期

### 3.1 列表

`GET /api/important-dates?tag=证件&page=1&page_size=50`

响应项：

```json
{
  "id": "date_123",
  "title": "护照到期",
  "date": "2026-12-01",
  "date_type": "证件",
  "repeat_rule": "不重复",
  "note": "",
  "tags": ["证件"]
}
```

### 3.2 创建和更新

`POST /api/important-dates`

`PUT /api/important-dates/{id}`

请求：

```json
{
  "title": "护照到期",
  "date": "2026-12-01",
  "date_type": "证件",
  "repeat_rule": "不重复",
  "note": "",
  "tags": ["证件"]
}
```

规则：

- `title`、`date`、`date_type` 必填。
- `repeat_rule` 为空时按 `不重复` 处理。
- `repeat_rule` 只允许 `不重复`、`每年`、`每月`、`每周`。

### 3.3 删除

`DELETE /api/important-dates/{id}`

删除成功写入审计事件。

## 4. 账单和预算

### 4.1 账单列表

`GET /api/transactions?from=2026-07-01&to=2026-07-31&type=支出&category=餐饮&account=现金&tag=家庭&page=1&page_size=50`

响应项：

```json
{
  "id": "txn_123",
  "date": "2026-07-04",
  "time": "08:30",
  "type": "支出",
  "amount": "25.50",
  "category": "餐饮",
  "include_income": true,
  "include_budget": true,
  "ledger": "默认账本",
  "counterparty": "早餐店",
  "account": "现金",
  "tags": ["日常"],
  "note": ""
}
```

### 4.2 创建和更新账单

`POST /api/transactions`

`PUT /api/transactions/{id}`

请求：

```json
{
  "date": "2026-07-04",
  "time": "08:30",
  "type": "支出",
  "amount": "25.50",
  "category": "餐饮",
  "include_income": true,
  "include_budget": true,
  "ledger": "默认账本",
  "counterparty": "早餐店",
  "account": "现金",
  "tags": ["日常"],
  "note": ""
}
```

必填：日期、时间、类型、金额、分类、计入收支、计入预算、所属账本。

规则：

- `type` 只允许 `收入`、`支出`。
- `amount` 必须能解析为大于 0 的金额。
- `include_budget=true` 只对支出账单参与预算统计。

### 4.3 删除账单

`DELETE /api/transactions/{id}`

删除成功写入审计事件。

### 4.4 统计

`GET /api/transactions/summary?from=2026-07-01&to=2026-07-31`

响应：

```json
{
  "income": "10000.00",
  "expense": "3200.00",
  "balance": "6800.00",
  "by_category": [
    {
      "category": "餐饮",
      "expense": "1200.00",
      "ratio": 0.375
    }
  ]
}
```

只有 `include_income=true` 的账单参与收入、支出和余额统计。

### 4.5 预算

`GET /api/budgets?month=2026-07`

`POST /api/budgets`

`PUT /api/budgets/{id}`

`DELETE /api/budgets/{id}`

请求：

```json
{
  "month": "2026-07",
  "category": "餐饮",
  "amount": "1500.00"
}
```

响应项：

```json
{
  "id": "budget_123",
  "month": "2026-07",
  "category": "餐饮",
  "amount": "1500.00",
  "used": "1200.00",
  "remaining": "300.00",
  "usage_ratio": 0.8,
  "overspent": false
}
```

预算使用只统计同月份、同分类、类型为 `支出` 且 `include_budget=true` 的账单。

## 5. Excel

### 5.1 下载模板

`GET /api/transactions/template.xlsx`

返回 `.xlsx` 文件，首个工作表表头固定为：

```text
日期, 时间, 类型, 金额, 分类, 计入收支, 计入预算, 所属账本, 对象, 账户, 标签, 备注
```

### 5.2 导出

`GET /api/transactions/export.xlsx?from=2026-07-01&to=2026-07-31`

返回 `.xlsx` 文件，表头与模板一致，内容符合筛选条件。

### 5.3 导入

`POST /api/transactions/import.xlsx`

请求类型：`multipart/form-data`，字段名 `file`。

限制：

- 只接受 `.xlsx`。
- 文件大小最大 5MB。
- 只读取第一个工作表。
- 最大 5000 行数据。
- 表头必须与模板一致。

成功响应：

```json
{
  "imported": 128
}
```

失败响应：

```json
{
  "error": {
    "code": "validation_failed",
    "message": "Excel 校验失败",
    "details": [
      {
        "row": 12,
        "column": "金额",
        "reason": "金额必须大于 0"
      }
    ]
  }
}
```

导入失败不得写入任何账单，不得回显整行原始内容。

## 6. 决策

### 6.1 列表

`GET /api/decisions?status=进行中&tag=工作&page=1&page_size=50`

响应项：

```json
{
  "id": "decision_123",
  "title": "是否搬家",
  "background": "通勤时间过长",
  "final_choice": "",
  "status": "进行中",
  "review_date": null,
  "review_note": "",
  "options": [
    {
      "id": "option_1",
      "name": "搬近公司",
      "pros": "节省时间",
      "cons": "租金更高",
      "note": ""
    }
  ],
  "tags": ["生活"]
}
```

### 6.2 创建和更新

`POST /api/decisions`

`PUT /api/decisions/{id}`

请求：

```json
{
  "title": "是否搬家",
  "background": "通勤时间过长",
  "final_choice": "",
  "status": "进行中",
  "review_date": null,
  "review_note": "",
  "options": [
    {
      "name": "搬近公司",
      "pros": "节省时间",
      "cons": "租金更高",
      "note": ""
    }
  ],
  "tags": ["生活"]
}
```

规则：

- `title` 必填。
- `status` 只允许 `进行中`、`待复盘`、`已归档`。
- 每个候选方案的 `name` 必填。
- `review_date <= today` 且未归档的记录查询时可展示为 `待复盘`。

### 6.3 删除

`DELETE /api/decisions/{id}`

删除成功写入审计事件。

## 7. 标签

标签由业务保存接口随对象一起写入。首版不提供独立一级页面。

`GET /api/tags?query=家`

响应：

```json
{
  "items": [
    {
      "id": "tag_123",
      "name": "家庭"
    }
  ]
}
```

标签名去掉首尾空格后不能为空，重复标签复用已有记录。

## 8. 审计

`GET /api/audit-events?page=1&page_size=50`

首版可仅供设备管理或后续设置入口使用。

响应项：

```json
{
  "id": "audit_123",
  "event_type": "login_success",
  "occurred_at": "2026-07-04T12:00:00Z",
  "client_ip": "127.0.0.1",
  "device_id": "dev_123",
  "resource_type": "",
  "resource_id": "",
  "result": "success",
  "reason": ""
}
```
