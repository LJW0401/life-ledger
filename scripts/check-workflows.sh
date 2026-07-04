#!/usr/bin/env sh
# Checks workflow details that actionlint cannot infer from shell semantics.
set -eu

ci=".github/workflows/ci.yml"

if ! grep -Fq 'npm --prefix web exec -- playwright install --with-deps chromium' "$ci"; then
  echo "$ci must pass Playwright flags through npm exec with --" >&2
  exit 1
fi

if grep -Fq 'npm --prefix web exec playwright install --with-deps chromium' "$ci"; then
  echo "$ci contains npm exec without -- before Playwright flags" >&2
  exit 1
fi
