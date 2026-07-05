#!/usr/bin/env sh
# Checks workflow details that actionlint cannot infer from shell semantics.
set -eu

ci=".github/workflows/ci.yml"
release=".github/workflows/release.yml"

if ! grep -Fq 'npm --prefix web exec -- playwright install --with-deps chromium' "$ci"; then
  echo "$ci must pass Playwright flags through npm exec with --" >&2
  exit 1
fi

if grep -Fq 'npm --prefix web exec playwright install --with-deps chromium' "$ci"; then
  echo "$ci contains npm exec without -- before Playwright flags" >&2
  exit 1
fi

if ! grep -Eq "^[[:space:]]+prerelease: \\\$\\{\\{ contains\\(github\\.ref_name, 'preview'\\) \\}\\}$" "$release"; then
  echo "$release must publish tags containing preview as GitHub prereleases" >&2
  exit 1
fi

if ! grep -Eq "^[[:space:]]+make_latest: \\\$\\{\\{ contains\\(github\\.ref_name, 'preview'\\) && 'false' \\|\\| 'true' \\}\\}$" "$release"; then
  echo "$release must keep preview tags out of GitHub Latest Release" >&2
  exit 1
fi
