# OCI Bassh

SSH to OCI compute hosts through OCI Bastion without making the operator think
about sessions, OCIDs, or temporary bastion hostnames.

![OCI Bassh terminal demo](docs/assets/oci-bassh-demo.gif)

`oci-bassh` is the small front-door CLI for the OCI SSH workflow:

- explain why `ssh <host>` will or will not work
- track a compute instance from Terraform outputs
- create or renew the OCI Bastion managed SSH session
- write the VM-facing SSH config so the final command is still `ssh <host>`
- return stable JSON for agents and automation

It wraps the lower-level `oci-context` and `bastion-session` CLIs with one
operator-friendly Cobra command surface.

## Agent Support

`oci-bassh` is agent-ready, not Codex-only. The CLI exposes stable JSON output,
predictable exit codes, and machine-readable envelopes for automation.

Portable agent guidance lives in:

- `skills/`: runtime-neutral workflow instructions for `oci-context` and
  `bastion-session`
- `agents/`: adapter metadata and quick prompts for different agent runtimes
- `.codex-plugin/`: Codex packaging for the same portable skills

The Codex plugin is one distribution adapter. The command contracts and skills
are intended to be reusable by any agent that can run shell commands and read
JSON.

## Install

Homebrew is the preferred install path:

```bash
brew tap adrianmross/tap
brew install oci-bassh
```

The Homebrew binary is installed at:

```bash
/opt/homebrew/bin/oci-bassh
```

Source install is also supported:

```bash
curl -sSL https://raw.githubusercontent.com/adrianmross/oci-bassh/main/install.sh | bash
```

By default the installer writes to `/usr/local/bin`. Override it with
`PREFIX`:

```bash
PREFIX="$HOME/.local" curl -sSL https://raw.githubusercontent.com/adrianmross/oci-bassh/main/install.sh | bash
```

## Quickstart

Check local dependencies and context:

```bash
oci-bassh doctor
```

Explain the host-facing SSH path:

```bash
oci-bassh explain my-vps-01
```

Create or reuse the bastion session and update SSH config:

```bash
oci-bassh ensure my-vps-01
```

Connect to the compute instance, not to the bastion alias:

```bash
ssh my-vps-01
```

## Host Model

The durable target is the compute host alias you already type, such as
`my-vps-01`.

`oci-bassh` keeps the internal bastion jump host fresh, but the operator-facing
target remains:

```sshconfig
Host my-vps-01
  HostName 10.0.1.25
  User cloud-user
  ProxyJump my-bastion
```

That means `ssh my-vps-01` goes through OCI Bastion while still landing directly
on the compute instance.

## Common Commands

```bash
oci-bassh doctor
oci-bassh check
oci-bassh inspect my-vps-01
oci-bassh explain my-vps-01
oci-bassh repair --ensure my-vps-01
oci-bassh track my-vps-01 ./tf
oci-bassh ensure my-vps-01
oci-bassh ssh --dry-run my-vps-01
oci-bassh paths -o json
oci-bassh version -o json
oci-bassh upgrade
oci-bassh contract-check
```

The longer aliases remain available when the caller wants names that describe
the underlying operation exactly:

```bash
oci-bassh track-from-terraform my-vps-01 ./tf
oci-bassh ensure-target my-vps-01
```

## Paths

Use `paths` when scripts or agents need to know where local state lives:

```bash
oci-bassh paths -o json
```

Typical paths:

- Homebrew binary: `/opt/homebrew/bin/oci-bassh`
- Source install binary: `/usr/local/bin/oci-bassh`
- SSH include: `~/.ssh/config.d/bastion-session`
- Bastion session cache: `~/.cache/bastion-session/state.json`
- Tracked targets: `~/.cache/bastion-session/tracked-targets.json`
- Current OCI context config: `~/.oci-context/config.yml`

Your `~/.ssh/config` should include the managed fragment:

```sshconfig
Include ~/.ssh/config.d/bastion-session
```

## Output Contract

Stable automation output is JSON. Agents should prefer `-o json`,
`--output json`, or command-specific JSON flags instead of parsing human text.

`doctor` is tolerant and always emits a result envelope. `check` is strict and
returns non-zero when dependencies are unhealthy.

`explain <host>` wraps:

```bash
bastion-session explain <host> -o json
```

and returns the downstream result inside the stable `oci-bassh` envelope.

For ordinary inspection, the skills prefer:

```bash
oci-context status --cached -o json
oci-context auth show --output json
oci-context auth ensure --output json
```

They avoid `oci-context export` unless the task is explicitly to export shell
environment settings or a context handoff.

## Shell Completions

`oci-bassh` is built with Cobra and can generate completions:

```bash
oci-bassh completion zsh > "${fpath[1]}/_oci-bassh"
oci-bassh completion bash > /usr/local/etc/bash_completion.d/oci-bassh
oci-bassh completion fish > ~/.config/fish/completions/oci-bassh.fish
```

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/oci-bassh
go test -tags=e2e ./...
```

The default Go test suite includes strict hermetic command-contract coverage.
The `e2e` tagged tests use fake `oci` and `ssh` shims while exercising real
`oci-context` and `bastion-session` binaries. Override those binaries with:

```bash
OCI_CONTEXT_BIN=/path/to/oci-context \
BASTION_SESSION_BIN=/path/to/bastion-session \
go test -tags=e2e ./...
```

JSON schema files live under `schemas/`. The Go tests validate top-level
compatibility for the public command contract.
