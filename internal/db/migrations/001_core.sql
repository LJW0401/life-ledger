CREATE TABLE important_dates (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  date TEXT NOT NULL,
  date_type TEXT NOT NULL,
  repeat_rule TEXT NOT NULL DEFAULT '不重复',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (repeat_rule IN ('不重复', '每年', '每月', '每周'))
);

CREATE TABLE transactions (
  id TEXT PRIMARY KEY,
  occurred_date TEXT NOT NULL,
  occurred_time TEXT NOT NULL,
  type TEXT NOT NULL,
  amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
  category TEXT NOT NULL,
  include_income INTEGER NOT NULL CHECK (include_income IN (0, 1)),
  include_budget INTEGER NOT NULL CHECK (include_budget IN (0, 1)),
  ledger TEXT NOT NULL,
  counterparty TEXT NOT NULL DEFAULT '',
  account TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (type IN ('收入', '支出'))
);

CREATE TABLE budgets (
  id TEXT PRIMARY KEY,
  month TEXT NOT NULL,
  category TEXT NOT NULL,
  amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (month, category)
);

CREATE TABLE decisions (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  background TEXT NOT NULL DEFAULT '',
  final_choice TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  review_date TEXT,
  review_note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  CHECK (status IN ('进行中', '待复盘', '已归档'))
);

CREATE TABLE decision_options (
  id TEXT PRIMARY KEY,
  decision_id TEXT NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  pros TEXT NOT NULL DEFAULT '',
  cons TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL
);

CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);

CREATE TABLE entity_tags (
  entity_type TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
  created_at TEXT NOT NULL,
  PRIMARY KEY (entity_type, entity_id, tag_id),
  CHECK (entity_type IN ('important_date', 'transaction', 'decision'))
);

CREATE TABLE audit_events (
  id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  occurred_at TEXT NOT NULL,
  client_ip TEXT NOT NULL DEFAULT '',
  device_id TEXT,
  user_agent TEXT NOT NULL DEFAULT '',
  resource_type TEXT NOT NULL DEFAULT '',
  resource_id TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  CHECK (result IN ('success', 'failure'))
);
