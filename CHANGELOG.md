# Changelog

All notable changes to this project will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to **Semantic Versioning**.

## [Unreleased]
- (placeholder for upcoming changes)

## [v0.1.0] - 2025-09-05
### Added
- Initial public release.
- Dual executables: `fit` (libfido2 hardware) & `fit-hello` (Windows Hello / platform + external selectors).
- Commands:
  - Common: `list`, `test`, `add-passkey`, `auth`.
  - `fit` only: `set-pin`, `reset`.
  - `fit-hello` only: `delete-passkey`.
- JSON output mode across key operations.
- Random challenge generation with both `challengeHex` and `challengeB64` in auth / passkey creation outputs.
- Shared `internal/chal` helper for cryptographically secure challenges + encoders.
- Enhanced PIN error messaging (policy, mismatch, existing PIN hints).
- Documentation overhaul separating the two CLIs and detailing workflows.

### Notes
- Windows Hello cannot manage PIN or perform device reset.
- Transient credentials via `--create` (non-resident) are supported in `auth` (hardware path).

[v0.1.0]: https://example.com/tag/v0.1.0  
