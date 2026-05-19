# AGENTS

Operational guide for maintaining OCI Bastion Hopper.

## Scope
- Keep instructions generic and environment-agnostic.
- Do not include personal hostnames, usernames, private IPs, proxy aliases, or absolute machine-specific paths.

## Demo Assets
- The README terminal capture is generated from `docs/demo/oci-hop.tape` with VHS.
- Keep the README focused on product usage; implementation details for regenerating the capture belong here.
- Use fictional examples in demo assets, currently `my-vps-01`, `my-bastion`, `cloud-user`, and `10.0.1.25`.
- After changing demo scripts or tapes, run:
  - `vhs validate docs/demo/oci-hop.tape`
  - `vhs docs/demo/oci-hop.tape`

## Agent Contract
- Use JSON output for automation wherever the CLI supports it.
- Prefer `-o json`, `--output json`, or command-specific JSON flags over parsing human text.
- `doctor` is tolerant and should emit a result envelope even when dependencies are unhealthy.
- `check` is strict and returns non-zero when diagnostics are unhealthy.
- `explain <host>` wraps `bastion-session explain <host> -o json` inside the stable `oci-hop` envelope.
- Prefer `oci-context status --cached -o json`, `oci-context auth show --output json`, and `oci-context auth ensure --output json` for ordinary inspection.
- Avoid `oci-context export` unless the task is explicitly to export shell environment settings or a context handoff.
- Use `hop paths -o json` when scripts or agents need local state paths.
- Use `hop contract-check` to verify downstream JSON command contracts.
- Treat JSON field names as stable contract. Prefer additive fields and document any breaking output change before relying on it in workflows or scripts.
- Keep `skills/` and `agents/` runtime-neutral. `.codex-plugin/` is a Codex adapter over the portable agent instructions, not the source of truth.
- Preferred validation commands are `make fmt`, `make vet`, `make test`, `make lint-workflows`, and `make validate-workflows`.
