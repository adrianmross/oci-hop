# OCI Bassh

Codex plugin workflows for SSH access to OCI compute hosts through OCI Bastion
managed SSH sessions.

The plugin packages two skills:

- `oci-context`: agent-safe OCI context and auth inspection.
- `bastion-session`: tracking, ensuring, and diagnosing Bastion-backed SSH hosts.

The helper script gives agents one JSON-producing surface for common flows:

```bash
python3 scripts/oci_bassh.py doctor
python3 scripts/oci_bassh.py track-from-terraform vmordws02 ./tf
python3 scripts/oci_bassh.py ensure-target vmordws02
python3 scripts/oci_bassh.py contract-check
```

For ordinary inspection, the skills prefer `oci-context status --cached -o json`,
`oci-context auth show --output json`, and `oci-context auth ensure --output json`.
They avoid `oci-context export` unless the task is explicitly to export shell
environment settings or a context handoff.

## Validation

```bash
python3 -m py_compile scripts/*.py
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
