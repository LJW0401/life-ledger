#!/usr/bin/env sh
set -eu

tmp_dir="$(mktemp -d)"
data_dir="$tmp_dir/data"
mkdir -p "$data_dir"
chmod 700 "$data_dir"

cat > "$tmp_dir/config.toml" <<'CONFIG'
[server]
host = "127.0.0.1"
port = 18080

[data]
dir = "__DATA_DIR__"
database = "life-ledger.db"

[auth]
username = "admin"
password_hash = "$2a$12$d56nYheYVwS2GgyLM1HtbO6x3CemahxbtPZGFfly.X4bQoGIJJTYy"
session_secret = "01234567890123456789012345678901"
CONFIG

sed -i "s#__DATA_DIR__#$data_dir#g" "$tmp_dir/config.toml"
chmod 600 "$tmp_dir/config.toml"

exec go run ./cmd/server --config "$tmp_dir/config.toml"
