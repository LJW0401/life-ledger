<!--
release.template.md - Copy this file to release.md before pushing a release tag.
Keep release-version exactly equal to the pushed git tag; Release CI checks it.
-->
<!-- release-version: <VERSION> -->

# Release <VERSION>

> Baseline: `<PREV_VERSION>` · Compare: `<PREV_VERSION>...<VERSION>`
> Delete this line for the first release.

## Highlights

- Describe the user-visible theme of this release.

## Fixes

- Describe user-visible fixes.

## Operational Notes

- The release workflow uploads Linux amd64 and Linux arm64 single-file binaries.
- Deploy by replacing the existing `life-ledger` binary and keeping `config.toml` beside it.
- If `config.example.toml` changed, review whether your production `config.toml` needs the new keys.
