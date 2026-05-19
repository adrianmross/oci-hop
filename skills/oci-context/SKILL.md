---
name: oci-context
description: Use when managing OCI CLI context, auth readiness, profile/region/compartment status, daemon health, or preparing an OCI context for automation with the `oci-context` CLI. Prefer status/current/auth show over export unless the user explicitly wants shell environment export.
---

# OCI Context

Use this skill for `oci-context` workflows. Prefer machine-readable commands and avoid parsing human text.

## Preferred Commands

- Current/config-only context inspection:
  ```bash
  oci-context status --cached -o json
  ```
- Auth readiness before OCI-dependent work:
  ```bash
  oci-context auth ensure --output json
  ```
- Auth metadata:
  ```bash
  oci-context auth show --output json
  ```
- Full health summary when available:
  ```bash
  oci-context doctor -o json
  ```

Use `oci-context current` when only the current context name is needed.

Do not use `oci-context export` for ordinary inspection. Use `export` only when the task is to produce shell environment settings or a directory/workspace context handoff.

## Non-Interactive Behavior

For agent runs, prefer non-interactive paths. If a command reports `login_required`, tell the user the exact login command or rerun with explicit permission for interactive login.

## Helper CLI

For a compact machine-readable health check:

```bash
hop doctor
```

`hop doctor` is tolerant and should still return JSON when dependencies
are unhealthy. Use `hop check` when an automation needs a non-zero exit
for unhealthy diagnostics.
