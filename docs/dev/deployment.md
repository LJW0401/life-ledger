# life-ledger 部署与恢复设计

> 创建日期：2026-07-04
> 状态：已完成
> 版本：v1.0
> 关联文档：`docs/dev/prd.md`、`docs/dev/sad.md`

本文定义首版 Linux 云服务器部署、Caddy 反代、文件权限、备份和手动恢复流程。部署目标保持为单个二进制文件 + `config.toml` + 数据目录。

## 1. 部署目录

推荐目录：

```text
/opt/life-ledger/
  life-ledger
  config.toml
  data/
    life-ledger.db
  backups/
```

权限要求：

| 路径 | 权限 | 说明 |
|------|------|------|
| `/opt/life-ledger/` | `750` | 服务运行目录。 |
| `life-ledger` | `750` | 可执行文件。 |
| `config.toml` | `600` | 包含 `password_hash` 和 `session_secret`。 |
| `data/` | `700` | SQLite 数据目录。 |
| `data/life-ledger.db` | `600` | SQLite 数据库。 |
| `backups/` | `700` | 本地备份目录。 |

应用启动时必须检查 `config.toml`、数据目录和已存在数据库文件权限；权限过宽时启动失败。

## 2. 配置文件

默认启动命令：

```bash
./life-ledger
```

默认读取二进制文件所在目录下的 `config.toml`。配置中的相对 `data.dir` 和 `backup.dir` 也按 `config.toml` 所在目录解析。特殊场景可指定：

```bash
./life-ledger --config /opt/life-ledger/config.toml
```

配置示例：

```toml
[server]
host = "127.0.0.1"
port = 8080

[data]
dir = "./data"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = "$2a$12$..."
session_secret = "replace-with-at-least-32-random-bytes"
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

禁止配置明文密码字段。若出现 `password` 这类明文密码配置，应用应启动失败并提示改用 `password_hash`。

## 3. 安全配置辅助命令

生成密码哈希：

```bash
./life-ledger hash-password
```

生成会话密钥：

```bash
./life-ledger generate-secret
```

初始化本地配置：

```bash
./life-ledger init-config
```

`init-config` 默认在二进制文件所在目录创建 `config.toml`、`data/` 和 `backups/`，并输出一次性初始密码；若配置文件已存在则拒绝覆盖。`hash-password` 和 `generate-secret` 只输出配置值，不启动 HTTP 服务、不打开数据库、不修改配置文件。

## 4. Caddy 反向代理

Caddy 负责公网 HTTPS、域名和反向代理；应用负责登录、Cookie、CSRF、设备会话和 API 访问控制。

Caddyfile 示例：

```caddyfile
life.example.com {
  reverse_proxy 127.0.0.1:8080
}
```

应用默认监听 `127.0.0.1`，不直接暴露公网端口。若显式改为公网地址，必须同时在服务器防火墙层限制访问来源。

可信反代 IP 默认只包含 `127.0.0.1`。只有请求来自可信反代地址时，应用才读取 `X-Forwarded-For` 中的第一个 IP；否则忽略客户端伪造的 `X-Forwarded-For` 和 `X-Real-IP`。

## 5. systemd 示例

`/etc/systemd/system/life-ledger.service`：

```ini
[Unit]
Description=life-ledger
After=network.target

[Service]
Type=simple
User=life-ledger
Group=life-ledger
WorkingDirectory=/opt/life-ledger
ExecStart=/opt/life-ledger/life-ledger
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now life-ledger
```

查看日志：

```bash
journalctl -u life-ledger -f
```

日志不得输出明文密码、`password_hash`、`session_secret`、session token、Cookie、CSRF token 或 Excel 原始内容。

## 6. 构建和发布包

本地构建流程：

```bash
npm --prefix web install
go test ./...
make build
```

发布包至少包含：

```text
life-ledger
config.example.toml
README.md
```

正式服务器上的 `config.toml` 不应被发布包覆盖。

## 7. 备份

执行：

```bash
./life-ledger backup
```

备份命令行为：

- 不启动 HTTP 服务。
- 不修改数据库。
- 读取当前 `config.toml`、SQLite 数据库和 schema 版本。
- 在 `backups/` 下生成本地备份包。

备份包内容：

```text
life-ledger-backup-20260704-120000/
  life-ledger.db
  config.toml
  backup-meta.json
```

`backup-meta.json`：

```json
{
  "app_version": "0.1.0",
  "backup_time": "2026-07-04T12:00:00Z",
  "schema_version": 3,
  "files": ["life-ledger.db", "config.toml"]
}
```

备份包包含全部个人数据和登录能力，不默认加密，不默认上传云端。用户需要自行妥善保存，必要时在服务器外部额外加密。

## 8. 手动恢复

首版不提供 `restore` 命令。恢复流程：

1. 停止服务。

```bash
sudo systemctl stop life-ledger
```

2. 备份当前损坏或待替换目录，避免覆盖后无法回退。

```bash
sudo cp -a /opt/life-ledger/data /opt/life-ledger/data.before-restore
sudo cp -a /opt/life-ledger/config.toml /opt/life-ledger/config.before-restore.toml
```

3. 替换配置和数据库。

```bash
sudo cp /path/to/backup/config.toml /opt/life-ledger/config.toml
sudo cp /path/to/backup/life-ledger.db /opt/life-ledger/data/life-ledger.db
```

4. 修正所有者和权限。

```bash
sudo chown -R life-ledger:life-ledger /opt/life-ledger
sudo chmod 750 /opt/life-ledger
sudo chmod 600 /opt/life-ledger/config.toml
sudo chmod 700 /opt/life-ledger/data
sudo chmod 600 /opt/life-ledger/data/life-ledger.db
sudo chmod 700 /opt/life-ledger/backups
```

5. 启动服务并验证。

```bash
sudo systemctl start life-ledger
sudo systemctl status life-ledger
```

验证项：

- 浏览器能打开登录页。
- 登录成功后日期、账单、决策数据可读。
- `journalctl` 中没有敏感配置、Cookie 或 token。

## 9. 部署冒烟

发布前至少执行：

```bash
./life-ledger --config ./config.toml
curl -I http://127.0.0.1:8080/
curl -I http://127.0.0.1:8080/important-dates
```

检查：

- 服务只监听 `127.0.0.1:<port>`。
- `/` 和三个一级页面能返回前端入口。
- `/api/session` 未登录返回 401。
- 响应包含基础安全头。
- Caddy 域名 HTTPS 可访问，并由应用登录页保护。
