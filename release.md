<!--
release.md - Release notes consumed by GitHub Actions when a version tag is pushed.
Update release-version before tagging; CI fails if it does not equal the tag.
-->
<!-- release-version: v0.1.0 -->

# Release v0.1.0

Initial preview release for personal deployment.

## Highlights

- Single-binary Go web service with embedded React frontend.
- Single-user login with remembered devices and CSRF protection.
- Important dates, bills, budgets, and decisions persisted in SQLite.
- Excel bill import/export.
- Local backup command for SQLite snapshot plus `config.toml`.

## Operational Notes

- Release assets include Linux amd64 and Linux arm64 binaries.
- Put `config.toml` next to the binary; relative data and backup paths resolve from that config file.
- Use `./life-ledger init-config` for first local setup, then edit `config.toml` as needed.
