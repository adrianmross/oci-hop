# OCI Bassh

Codex plugin workflows for SSH access to OCI compute hosts through OCI Bastion
managed SSH sessions.

The plugin packages two skills:

- `oci-context`: agent-safe OCI context and auth inspection.
- `bastion-session`: tracking, ensuring, and diagnosing Bastion-backed SSH hosts.

The Go CLI gives agents one JSON-producing surface for common flows:

```bash
oci-bassh doctor
oci-bassh check
oci-bassh inspect vmordws02
oci-bassh repair --ensure vmordws02
oci-bassh track vmordws02 ./tf
oci-bassh ensure vmordws02
oci-bassh ssh --dry-run vmordws02
oci-bassh explain vmordws02
oci-bassh paths -o json
oci-bassh upgrade
oci-bassh version -o json
oci-bassh contract-check
```

From a checkout:

```bash
go run ./cmd/oci-bassh doctor
go run ./cmd/oci-bassh inspect vmordws02
go run ./cmd/oci-bassh track vmordws02 ./tf
go run ./cmd/oci-bassh ensure vmordws02
go run ./cmd/oci-bassh ssh vmordws02
go run ./cmd/oci-bassh explain vmordws02
go run ./cmd/oci-bassh paths
```

The longer aliases remain available when the caller wants names that describe
the underlying operation exactly:

```bash
oci-bassh track-from-terraform vmordws02 ./tf
oci-bassh ensure-target vmordws02
```

Use `doctor` for tolerant diagnostics that always produce JSON. Use `check`
for strict health gates where unhealthy dependencies should return a non-zero
exit status.

`explain <host>` wraps the downstream `bastion-session explain <host> -o json`
surface and returns the downstream result inside the stable `oci-bassh` command
result envelope.

`paths -o text` prints local config/cache paths as `key=value` lines.
`paths -o json` emits the same paths as a JSON contract.

`upgrade` is safe by default: it emits dry-run JSON with the installer command
instead of executing it. Add `--run` only when the installer should be executed.
`--prefix` and `--release` are translated into `PREFIX` and `VERSION` for the
installer.

`version` defaults to text output. Use `version -o json`, `version --json`, or
`--version --json` for machine-readable version details.

For ordinary inspection, the skills prefer `oci-context status --cached -o json`,
`oci-context auth show --output json`, and `oci-context auth ensure --output json`.
They avoid `oci-context export` unless the task is explicitly to export shell
environment settings or a context handoff.

## Validation

```bash
go test ./...
go vet ./...
python3 -m py_compile scripts/*.py
go build ./cmd/oci-bassh
python3 scripts/e2e_fake_cli.py
python3 scripts/e2e_real_binaries.py
```

`e2e_real_binaries.py` uses fake `oci` and `ssh` shims while exercising real
`oci-context` and `bastion-session` binaries. Override the binaries with:

```bash
OCI_CONTEXT_BIN=/path/to/oci-context \
BASTION_SESSION_BIN=/path/to/bastion-session \
python3 scripts/e2e_real_binaries.py
```

## Contracts

JSON schema files live under `schemas/`. The E2E scripts validate top-level
compatibility for:

- `oci-bassh doctor`
- `oci-bassh check`
- `oci-bassh inspect`
- `oci-bassh repair`
- `oci-bassh track`
- `oci-bassh ensure`
- `oci-bassh ssh --dry-run`
- `oci-bassh explain`
- `oci-bassh paths -o json`
- `oci-bassh upgrade`
- `oci-bassh version -o json`
- `oci-bassh contract-check`

The schemas cover the stable wrapper contract. Nested downstream payloads from
`oci-context` and `bastion-session` remain owned by those CLIs.
