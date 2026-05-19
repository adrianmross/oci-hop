# Agent Adapters

This directory contains portable metadata and prompts for agent runtimes that
can run shell commands and consume JSON.

The source operating guidance is intentionally runtime-neutral:

- `../skills/oci-context/SKILL.md`
- `../skills/bastion-session/SKILL.md`
- `generic.md`

Runtime-specific files such as `openai.yaml` and `claude.md` are adapters over
that shared contract. They should not introduce behavior that only works in one
agent unless the limitation is explicitly documented.

Codex packaging lives separately in `../.codex-plugin/`.
