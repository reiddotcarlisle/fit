# fit — FIDO Integration Tool

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](./LICENSE)

Two focused executables:

- `fit` — Talks directly to USB/NFC/BLE security keys using `go-libfido2`.
- `fit-hello` — Uses the Windows WebAuthn (Hello) API for platform & external authenticators.

Shared themes: list credentials/devices, diagnostics, create passkeys, perform assertions. PIN set/change and factory reset exist only in `fit` (hardware path).

## Layout

| Path              | Purpose                                      |
| ----------------- | -------------------------------------------- |
| `cmd/fit`       | libfido2 CLI (hardware keys)                 |
| `cmd/fit-hello` | Windows Hello CLI (platform/external via OS) |
| `internal/chal` | Shared challenge helper (random + encoders)  |

## Build

Quick builds:

PowerShell (Windows):
```pwsh
scripts/build.ps1
```

Bash (macOS / Linux / WSL):
```bash
scripts/build.sh
```

Make (any platform with make):
```bash
make build
```

Manual (Windows example):
```pwsh
go build -o bin/fit ./cmd/fit
go build -o bin/fit-hello ./cmd/fit-hello
Copy-Item lib/*.dll bin/
```

Notes:

- Runtime DLLs are versioned under `lib/` (so builds are reproducible). Build scripts automatically copy them to `bin/`.
- The `bin/` directory and generated executables are git‑ignored; run the build + copy after cloning.
- Only `fit` and `fit-hello` are required; any additional vendor helper binaries placed in `bin/` are optional.

## Command summary

### fit (hardware / libfido2)

- `list` — Enumerate attached FIDO2 devices.
- `info [--pin PIN] [--device N|--path PATH]` — Non‑destructive diagnostics (type, versions, options, retry count, resident key stats if PIN supplied).
- `set-pin --new NEW [--old OLD] [--device N|--path PATH]` — Set initial PIN or change an existing one.
- `reset [--device N|--path PATH]` — Factory reset (wipes credentials; irreversible).
- `add-passkey --rp RP_ID [--user USER] [--display NAME] [--resident|--no-resident] [--pin PIN] [--device N|--path PATH]` — Create resident (discoverable) or non‑resident credential.
- `auth --rp RP_ID [--pin PIN] [--cred-id-hex HEX|--cred-index N] [--create] [--device N|--path PATH]` — Perform assertion. `--create` first makes a transient non‑resident credential then asserts it.

### fit-hello (Windows Hello)

- `list [--rp RP]` — List platform credentials (filter by RP).
- `info [--rp RP]` — Light diagnostics + subset of credentials (capped for brevity).
- `add-passkey --rp RP [--user USER] [--display NAME] [--device] [--resident|--no-resident]` — Create passkey (platform by default, `--device` prefers external key).
- `auth --rp RP [--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N|--device]` — Perform assertion. If neither allow list nor credential chosen and `--device` set, Windows UI lets user pick an external key.
- `delete-passkey [--rp RP] (--cred-id-hex HEX|--cred-id-b64 B64URL|--cred-index N)` — Delete a platform credential.

## Device / credential selection

Hardware (`fit`):

- `--device N` index from `fit list`.
- `--path PATH` exact device path.

Windows Hello (`fit-hello`):

- Credential selection usually by `--cred-index` after `list`.
- Or specify a credential ID using `--cred-id-b64` (base64url) or `--cred-id-hex`.
- `--device` (in `fit-hello`) hints preference for external security keys.

## Output formats

Credential IDs:

- `fit`: hex (`credentialID`).
- `fit-hello`: base64url (`credID` in list output; `credentialID` in JSON operation output).

Random challenges (added for auditability & server integration testing):

- `auth` / `add-passkey` now include both `challengeHex` and `challengeB64` in JSON for `fit`.
- `fit-hello` JSON includes `challengeHex` and `challengeB64` (naming: `challengeHex` / `challengeB64`).
- Human-readable output prints both encodings.

Why expose both? Real WebAuthn flows send a server‑generated challenge to the client. Here the CLI generates cryptographically random 32 bytes; exposing both encodings lets you copy either into test harnesses or verify signature binding. (Note: libfido2 uses the challenge as client data hash input directly; Windows Hello path embeds the base64url in `clientDataJSON`.)

## JSON field reference (selected)

`fit auth` (JSON):

```json
{
	"backend": "libfido2",
	"rp": "example.com",
	"credentialID": "...hex...",
	"signature": "...hex...",
	"challengeHex": "...",
	"challengeB64": "...",
	"hmacSecret": "...optional...",
	"authDataCBOR": "...optional..."
}
```

`fit add-passkey` (JSON) includes: `resident`, `credentialID`, `challengeHex`, `challengeB64`.

`fit-hello auth` / `add-passkey` (JSON) include: `credentialID` (base64url), `challengeHex`, `challengeB64`, plus optional PRF output (`prfFirst`) when available.

## Examples

Hardware key (resident credential):

```pwsh
bin/fit set-pin --new 1234
bin/fit add-passkey --rp example.com --user you@example.com --pin 1234
bin/fit auth --rp example.com --cred-index 0 --pin 1234 --json
```

Transient credential assertion (non‑resident):

```pwsh
bin/fit auth --rp example.com --create --pin 1234 --json
```

Windows Hello (platform credential):

```pwsh
bin/fit-hello add-passkey --rp example.com --user you@example.com --json
bin/fit-hello auth --rp example.com --cred-index 0 --json
```

Windows Hello external key preference:

```pwsh
bin/fit-hello add-passkey --rp example.com --user you@example.com --device
bin/fit-hello auth --rp example.com --device
```

Delete a platform credential:

```pwsh
bin/fit-hello delete-passkey --rp example.com --cred-index 0
```

## Typical workflow cheat sheet

1. Enumerate hardware: `bin/fit list`
2. Set PIN (first time): `bin/fit set-pin --new 1234`
3. Create resident passkey: `bin/fit add-passkey --rp example.com --pin 1234`
4. Assert: `bin/fit auth --rp example.com --cred-index 0 --pin 1234`
5. Check capacity: `bin/fit info --pin 1234 --json`

## Limitations / notes

- Windows Hello cannot set/change PIN or factory reset; those are device operations (use `fit`).
- `fit reset` destroys all credentials on the selected authenticator (irreversible).
- Some authenticators require user presence (touch) and may throttle retries on PIN errors.
- Transient (`--create`) credentials are non‑resident; only valid for that immediate assertion.

## Troubleshooting

| Symptom                              | Explanation / Action                                                             |
| ------------------------------------ | -------------------------------------------------------------------------------- |
| "Operation was canceled by the user" | You dismissed the Windows Hello prompt. Re‑run and complete UI flow.            |
| PIN policy / invalid                 | Chosen PIN too short / not accepted. Try 6+ digits or vendor recommendations.    |
| PIN mismatch                         | Wrong old PIN; retry count decreases. Use `fit info` to see remaining retries. |
| No devices found                     | Use `bin/fit list`; ensure drivers and permissions.                            |
| No credentials for RP                | Create one with `add-passkey` first.                                           |

## Future ideas

- Shared JSON schema versioning.
- Optional export of attestation objects for verification.
- Integration tests harness.

## Roadmap (platform & capability variants)

Planned / potential sibling binaries following the `fit-*` pattern:

### Cross-platform
- `fit-soft` — Pure software FIDO2/WebAuthn emulator for CI (configurable algorithms, counters, UV flags).
- `fit-sim` — Deterministic simulation backend for reproducible test vectors (subset focus of `fit-soft`).
- `fit-passkey` — Unified platform authenticator abstraction (Windows Hello + future macOS/Linux APIs) when mature.

### Windows
- `fit-tpm` — TPM 2.0 attested key creation & inspection (leverages go-tpm / tpm2-tools) for attestation chain testing.
- (Existing) `fit-hello` — Windows WebAuthn API (platform & external authenticators UI prompts).

### macOS
- `fit-touchid` — Touch ID / Face ID assertions via LocalAuthentication (assertion-focused until broader APIs exposed).
- `fit-se` — Secure Enclave key provisioning + COSE/attestation export for verification experiments.

### Linux
- `fit-pam` — Enrollment & diagnostic helper for pam_u2f / pam_fido2 flows.
- `fit-tpm` (shared concept) — Same goals as Windows TPM variant where a discrete TPM is present.

### Rationale
Maintain a small, purpose-built binary per backend to avoid complex flag matrices while keeping a consistent UX (list / info / add-passkey / auth / delete / reset where applicable). Core `fit` remains the hardware (libfido2) tool; others layer platform or virtual backends.

## Credits

`fit` (hardware path) is powered by the excellent [libfido2](https://github.com/Yubico/libfido2) project from Yubico, accessed through the Go bindings at [github.com/keys-pub/go-libfido2](https://github.com/keys-pub/go-libfido2). Huge thanks to its maintainers and contributors for making high‑quality, interoperable FIDO2 tooling available.

Windows Hello functionality uses the Windows WebAuthn API via community Go wrappers.

## License

Released under the MIT License. See [LICENSE](./LICENSE).

---

Happy hacking.
