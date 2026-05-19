# Generic Agent Guide

Use `hop` when an agent needs to inspect, repair, or prepare SSH access
to an OCI compute host through OCI Bastion.

## Contract

- Prefer JSON output over human text.
- Treat `ok: false` as actionable diagnostic output, not malformed output.
- Treat non-zero exits from strict commands as workflow failure.
- Preserve the compute host as the user-facing SSH target. Generated bastion
  aliases are internal `ProxyJump` endpoints.

## Preferred Commands

```bash
hop <host>
hop doctor
hop check
hop inspect <host>
hop explain <host>
hop repair --ensure <host>
hop ensure <host>
hop ssh --dry-run <host>
hop paths -o json
hop version -o json
hop contract-check
```

Use lower-level commands when a task specifically asks for `oci-context` or
`bastion-session` behavior:

```bash
oci-context status --cached -o json
oci-context auth ensure --output json
bastion-session target list -o json
bastion-session ensure <host> -o json
```

## Auth And Interaction

Prefer non-interactive checks first. If OCI auth requires browser login or other
interactive work, report the exact command needed before attempting it unless
the user already approved interactive authentication.
