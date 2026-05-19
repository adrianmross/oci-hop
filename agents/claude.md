# OCI Bassh Agent Instructions

Use these instructions for OCI Bastion SSH workflows from any shell-capable
agent.

## When To Use

Use `oci-bassh` when the user asks to connect to, diagnose, repair, or explain
SSH access to an OCI compute host through OCI Bastion.

## Operating Rules

- Prefer `oci-bassh` for end-to-end host workflows.
- Prefer `oci-context` only for context/auth questions.
- Prefer `bastion-session` only for lower-level session or target management.
- Use JSON output whenever available.
- Do not parse human-readable output if JSON is available.
- Keep examples generic. Do not expose real hostnames, usernames, private IPs,
  bastion aliases, or OCI identifiers.

## Command Flow

```bash
oci-bassh doctor
oci-bassh inspect <host>
oci-bassh explain <host>
oci-bassh ensure <host>
oci-bassh ssh --dry-run <host>
```

Run `ssh <host>` only when the user wants an actual connection attempt.
