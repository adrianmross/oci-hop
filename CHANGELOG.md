# Changelog

## Unreleased

- Add tolerant `doctor` diagnostics and strict `check` health gates.
- Add `inspect` for cached/no-live host inspection and `repair` for fix flows.
- Add CLI version aggregation, shell completions, and `track --terraform-dir`.
- Extend JSON contracts and E2E coverage for diagnostics, repair, and import UX.

## v0.3.0

- Promote `oci-bassh` to a releaseable Go CLI.
- Add GoReleaser, installer, and release workflows.
- Add JSON contract schema files for CLI outputs.
- Update E2E to exercise the Go CLI against fake and real downstream CLIs.

## v0.2.0

- Add the `bin/oci-bassh` command wrapper.
- Add short helper aliases: `track`, `ensure`, and `ssh`.
- Extend E2E coverage for command aliases and SSH dry-run output.

## v0.1.0

- Add the `oci-bassh` Codex plugin manifest.
- Add `oci-context` and `bastion-session` skills for agent-safe OCI Bastion SSH workflows.
- Add deterministic helper commands for doctor, target tracking, ensure, and contract checks.
- Add fake-CLI and real-binary E2E validation.
