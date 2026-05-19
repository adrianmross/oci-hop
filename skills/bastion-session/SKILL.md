---
name: bastion-session
description: Use when creating, refreshing, diagnosing, or connecting through OCI Bastion managed SSH sessions with the `bastion-session` CLI, including tracked OCI compute hosts such as vmordws02.
---

# Bastion Session

Use this skill for OCI Bastion managed SSH access. Prefer compute-host-facing targets over internal bastion aliases.

## Preferred Workflow

1. Ensure OCI auth is ready:
   ```bash
   oci-context auth ensure --output json
   ```
2. Inspect tracked targets:
   ```bash
   bastion-session target list -o json
   ```
3. Inspect a target without creating sessions:
   ```bash
   oci-bassh inspect <host>
   ```
4. Ensure the VM-facing SSH target:
   ```bash
   bastion-session ensure <host> -o json
   ```
5. Verify SSH config resolution:
   ```bash
   bastion-session ssh-config show <host> -o json
   ```

Use `ssh <host>` for the actual compute host connection. The generated `PROFILE-bastion` host is an internal ProxyJump endpoint, not the user-facing host target.

## Tracking a Target

Prefer Terraform outputs when available:

```bash
bastion-session target import <host> --terraform-outputs <dir-or-file>
```

Otherwise track explicitly:

```bash
bastion-session target track <host> \
  --instance-id <instance-ocid> \
  --private-ip <private-ip> \
  --bastion-id <bastion-ocid> \
  --user opc \
  --identity-file ~/.ssh/example.key
```

## Helper CLI

For an end-to-end ensure operation with one JSON result:

```bash
oci-bassh ensure-target <host>
```

For repairable setup drift, use:

```bash
oci-bassh repair --ensure <host>
```
