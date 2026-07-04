# life-ledger 发布回归记录

> 日期：2026-07-04
> 分支：`dev/life-ledger-implementation`
> 范围：首版发布硬化

## 1. 安全回归

- 认证保护：重要日期、账单、预算、决策、设备和审计 API 未登录访问返回 401。
- CSRF：所有写接口在登录但缺少 `X-CSRF-Token` 时返回 403。
- CORS：第三方 Origin 不返回 `Access-Control-Allow-Origin: *`。
- 安全响应头：应用路由统一设置 `X-Frame-Options`、`X-Content-Type-Options` 和 `Referrer-Policy`。
- 可信代理 IP：仅信任来自配置中反代地址的 `X-Forwarded-For`。
- 启动失败：缺少配置和端口占用均返回明确错误。

## 2. 性能回归

- 数据量：5000 条账单。
- 列表接口：`page_size` 超过 200 时截断为 200。
- 统计接口：收入、支出、余额和分类汇总改为 SQL 聚合。
- 门控：普通测试按 300ms 发布预算校验；race 测试只验证并发安全，使用更宽的测试预算避免检测开销误报。

## 3. 发布门控

已通过：

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

E2E 结果：8 个用例全部通过，覆盖登录、三页直刷、设备列表、日期、账单预算和决策归档。

## 4. 发布包检查

- 发布包包含：`life-ledger`、`config.example.toml`、`README.md`。
- 发布包不包含：真实 `config.toml`、SQLite 数据库、Excel 账单、密钥或备份包。
- 服务器部署后由 Caddy 提供 HTTPS，应用仅监听本机地址。
- 发布包冒烟：临时目录构建二进制，写入临时配置，启动后访问 `/important-dates` 成功返回前端页面。
