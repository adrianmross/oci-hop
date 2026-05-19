//go:build e2e

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRealLowerCLIBinariesContract(t *testing.T) {
	ociContext := resolveRequiredBinary(t, "OCI_CONTEXT_BIN", "oci-context")
	bastionSession := resolveRequiredBinary(t, "BASTION_SESSION_BIN", "bastion-session")
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	home := filepath.Join(tmp, "home")
	tfDir := filepath.Join(tmp, "tf")
	for _, dir := range []string{binDir, filepath.Join(home, ".oci-context"), filepath.Join(home, ".oci"), filepath.Join(home, ".ssh"), filepath.Join(home, ".cache", "bastion-session"), tfDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	binary := buildOCIBassh(t, tmp)
	symlinkExecutable(t, ociContext, filepath.Join(binDir, "oci-context"))
	symlinkExecutable(t, bastionSession, filepath.Join(binDir, "bastion-session"))
	writeRealBoundaryShims(t, binDir)
	writeRealBinaryState(t, home, tfDir)
	env := append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+home,
	)
	helper := []string{binary}

	checks := []struct {
		schema string
		args   []string
	}{
		{"oci-bassh-track.schema.json", append(helper, "track-from-terraform", "my-vps-01", tfDir)},
		{"oci-bassh-track.schema.json", append(helper, "track", "my-vps-01", tfDir)},
		{"oci-bassh-track.schema.json", append(helper, "track", "my-vps-01", "--terraform-dir", tfDir)},
		{"oci-bassh-ensure.schema.json", append(helper, "ensure-target", "my-vps-01")},
		{"oci-bassh-ensure.schema.json", append(helper, "ensure", "my-vps-01")},
		{"oci-bassh-ssh.schema.json", append(helper, "ssh", "--dry-run", "my-vps-01")},
		{"oci-bassh-doctor.schema.json", append(helper, "doctor")},
		{"oci-bassh-check.schema.json", append(helper, "check")},
		{"oci-bassh-inspect.schema.json", append(helper, "inspect", "my-vps-01")},
		{"oci-bassh-repair.schema.json", append(helper, "repair", "my-vps-01")},
		{"oci-bassh-repair.schema.json", append(helper, "repair", "--ensure", "my-vps-01")},
		{"oci-bassh-paths.schema.json", append(helper, "paths", "-o", "json")},
		{"oci-bassh-upgrade.schema.json", append(helper, "upgrade")},
		{"oci-bassh-version.schema.json", append(helper, "version", "-o", "json")},
		{"oci-bassh-version.schema.json", append(helper, "--version", "--json")},
	}

	for _, check := range checks {
		t.Run(check.schema, func(t *testing.T) {
			run := runCommandForTest(t, check.args, env)
			if run.code != 0 {
				t.Fatalf("command failed with %d\nstdout:\n%s\nstderr:\n%s", run.code, run.stdout, run.stderr)
			}
			payload := decodeObject(t, run.stdout)
			assertRequiredKeys(t, check.schema, payload)
			if payload["ok"] == false {
				t.Fatalf("real binary contract command returned ok=false\nstdout:\n%s", run.stdout)
			}
		})
	}
}

func resolveRequiredBinary(t *testing.T, envName, fallback string) string {
	t.Helper()
	if configured := os.Getenv(envName); configured != "" {
		return configured
	}
	path, err := exec.LookPath(fallback)
	if err != nil {
		t.Fatalf("%s must be set or %s must be on PATH when running -tags=e2e", envName, fallback)
	}
	return path
}

func symlinkExecutable(t *testing.T, src, dst string) {
	t.Helper()
	abs, err := filepath.Abs(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(abs, dst); err != nil {
		t.Fatal(err)
	}
}

func writeRealBoundaryShims(t *testing.T, binDir string) {
	t.Helper()
	writeExecutable(t, filepath.Join(binDir, "oci"), `#!/bin/sh
set -eu
if [ "$*" = "--version" ]; then
  printf '3.99.0\n'
  exit 0
fi
args=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --profile|--region|--auth|--config-file|--tenancy-id|--output)
      shift 2
      ;;
    *)
      args="${args}${args:+ }$1"
      shift
      ;;
  esac
done
case "$args" in
  "iam region-subscription list")
    printf '{"data":[{"is-home-region":true,"region-key":"PHX","region-name":"us-phoenix-1","status":"READY"}]}\n'
    ;;
  "bastion session list")
    printf '[]\n'
    ;;
  "bastion session create-managed-ssh"*)
    printf '{"id":"ocid1.bastionsession.oc1..fake","bastionId":"ocid1.bastion.oc1..b1","targetResourceId":"ocid1.instance.oc1..i1","targetResourceDetails":{"privateIpAddress":"10.0.1.25"},"lifecycleState":"ACTIVE","timeCreated":"2026-05-19T00:00:00Z","timeExpires":"2099-01-01T00:00:00Z"}\n'
    ;;
  "bastion session get"*)
    printf '{"id":"ocid1.bastionsession.oc1..fake","bastionId":"ocid1.bastion.oc1..b1","targetResourceId":"ocid1.instance.oc1..i1","targetResourceDetails":{"privateIpAddress":"10.0.1.25"},"lifecycleState":"ACTIVE","timeCreated":"2026-05-19T00:00:00Z","timeExpires":"2099-01-01T00:00:00Z"}\n'
    ;;
  *)
    printf 'unexpected oci %s\n' "$args" >&2
    exit 2
    ;;
esac
`)
	writeExecutable(t, filepath.Join(binDir, "ssh"), `#!/bin/sh
set -eu
case "$*" in
  "-G my-vps-01")
    printf 'user cloud-user\n'
    printf 'hostname 10.0.1.25\n'
    printf 'proxyjump my-bastion\n'
    printf 'identityfile ~/.ssh/fake.key\n'
    ;;
  "-V")
    printf 'OpenSSH_9.9\n'
    ;;
  *)
    printf 'unexpected ssh %s\n' "$*" >&2
    exit 2
    ;;
esac
`)
}

func writeRealBinaryState(t *testing.T, home, tfDir string) {
	t.Helper()
	mustWrite(t, filepath.Join(home, ".ssh", "id_ed25519.pub"), "ssh-ed25519 AAA fake\n")
	mustWrite(t, filepath.Join(home, ".ssh", "id_ed25519"), "private\n")
	mustWrite(t, filepath.Join(home, ".oci-context", "config.yml"), `options:
  oci_config_path: `+filepath.Join(home, ".oci", "config")+`
  socket_path: `+filepath.Join(home, ".oci-context", "daemon.sock")+`
contexts:
  - name: dev
    profile: DEFAULT
    auth_method: api_key
    tenancy_ocid: ocid1.tenancy.oc1..t
    compartment_ocid: ocid1.compartment.oc1..c
    region: us-phoenix-1
    user: ocid1.user.oc1..u
current_context: dev
`)
	outputs := map[string]map[string]string{
		"bastion_id":  {"value": "ocid1.bastion.oc1..b1"},
		"instance_id": {"value": "ocid1.instance.oc1..i1"},
		"private_ip":  {"value": "10.0.1.25"},
	}
	raw, err := json.Marshal(outputs)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(tfDir, "outputs.json"), string(raw))
	mustWrite(t, filepath.Join(home, ".cache", "bastion-session", "current-bastion.json"), `{"id":"ocid1.bastion.oc1..b1","name":"my-bastion","region":"us-phoenix-1","profile":"DEFAULT","auth_method":"api_key","context_name":"dev","source":"test","selected_at":"2026-05-19T00:00:00Z"}`)
}

func mustWrite(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
