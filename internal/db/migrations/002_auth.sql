CREATE TABLE device_sessions (
  id TEXT PRIMARY KEY,
  device_name TEXT NOT NULL,
  token_hash TEXT NOT NULL UNIQUE,
  csrf_token_hash TEXT NOT NULL,
  user_agent TEXT NOT NULL DEFAULT '',
  first_login_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  last_seen_ip TEXT NOT NULL DEFAULT '',
  expires_at TEXT NOT NULL,
  revoked_at TEXT
);

CREATE TABLE login_failures (
  id TEXT PRIMARY KEY,
  username TEXT NOT NULL,
  client_ip TEXT NOT NULL,
  failure_count INTEGER NOT NULL,
  window_started_at TEXT NOT NULL,
  last_failed_at TEXT NOT NULL,
  locked_until TEXT
);
