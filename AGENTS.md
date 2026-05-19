# AGENTS

Operational guide for maintaining `oci-bassh`.

## Scope
- Keep instructions generic and environment-agnostic.
- Do not include personal hostnames, usernames, private IPs, proxy aliases, or absolute machine-specific paths.

## Demo Assets
- The README terminal capture is generated from `docs/demo/oci-bassh.tape` with VHS.
- Keep the README focused on product usage; implementation details for regenerating the capture belong here.
- Use fictional examples in demo assets, currently `my-vps-01`, `my-bastion`, `cloud-user`, and `10.0.1.25`.
- After changing demo scripts or tapes, run:
  - `vhs validate docs/demo/oci-bassh.tape`
  - `vhs docs/demo/oci-bassh.tape`

## Agent Contract
- Use JSON output for automation wherever the CLI supports it.
- Treat JSON field names as stable contract. Prefer additive fields and document any breaking output change before relying on it in workflows or scripts.
- Keep `skills/` and `agents/` runtime-neutral. `.codex-plugin/` is a Codex adapter over the portable agent instructions, not the source of truth.
- Preferred validation commands are `make fmt`, `make vet`, `make test`, `make lint-workflows`, and `make validate-workflows`.
