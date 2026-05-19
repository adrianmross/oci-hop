package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var requiredKeys = map[string][]string{
	"oci-bassh-doctor.schema.json":         {"ok", "tools", "versions", "oci_context", "bastion_doctor", "targets"},
	"oci-bassh-check.schema.json":          {"ok", "tools", "versions", "oci_context", "bastion_doctor", "targets"},
	"oci-bassh-inspect.schema.json":        {"ok", "host", "versions", "oci_status", "auth", "bastion_doctor", "ssh_config", "ssh_effective"},
	"oci-bassh-repair.schema.json":         {"ok", "host", "repair", "ensure_requested", "connect_command"},
	"oci-bassh-ensure.schema.json":         {"ok", "host", "auth", "ensure", "ssh_config", "connect_command"},
	"oci-bassh-track.schema.json":          {"ok", "host", "track", "target"},
	"oci-bassh-ssh.schema.json":            {"ok", "host", "auth", "ensure", "ssh_command"},
	"oci-bassh-contract-check.schema.json": {"ok", "checks"},
	"oci-bassh-explain.schema.json":        {"ok", "host", "explain"},
	"oci-bassh-paths.schema.json":          {"ok", "paths"},
	"oci-bassh-upgrade.schema.json":        {"ok", "dry_run", "command"},
	"oci-bassh-version.schema.json":        {"ok", "version", "commit", "date"},
}

type commandRun struct {
	stdout string
	stderr string
	code   int
}

func TestSchemasAreValidJSON(t *testing.T) {
	for _, path := range mustGlob(t, filepath.Join(repoRoot(t), "schemas", "*.schema.json")) {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatalf("schema is not valid JSON: %v", err)
			}
		})
	}
}

func TestHermeticCLIContract(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	binary := buildOCIBassh(t, tmp)
	writeHermeticShims(t, binDir)
	env := append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	helper := []string{binary}

	checks := []struct {
		schema string
		args   []string
	}{
		{"oci-bassh-doctor.schema.json", append(helper, "doctor")},
		{"oci-bassh-check.schema.json", append(helper, "check")},
		{"oci-bassh-inspect.schema.json", append(helper, "inspect", "my-vps-01")},
		{"oci-bassh-repair.schema.json", append(helper, "repair", "my-vps-01")},
		{"oci-bassh-repair.schema.json", append(helper, "repair", "--ensure", "my-vps-01")},
		{"oci-bassh-ensure.schema.json", append(helper, "ensure-target", "my-vps-01")},
		{"oci-bassh-ensure.schema.json", append(helper, "ensure", "my-vps-01")},
		{"oci-bassh-track.schema.json", append(helper, "track-from-terraform", "my-vps-01", tmp)},
		{"oci-bassh-track.schema.json", append(helper, "track", "my-vps-01", tmp)},
		{"oci-bassh-track.schema.json", append(helper, "track", "my-vps-01", "--terraform-dir", tmp)},
		{"oci-bassh-ssh.schema.json", append(helper, "ssh", "--dry-run", "my-vps-01")},
		{"oci-bassh-ssh.schema.json", append(helper, "ssh", "--dry-run", "my-vps-01", "-p", "2222")},
		{"oci-bassh-explain.schema.json", append(helper, "explain", "my-vps-01")},
		{"oci-bassh-paths.schema.json", append(helper, "paths", "-o", "json")},
		{"oci-bassh-upgrade.schema.json", append(helper, "upgrade")},
		{"oci-bassh-version.schema.json", append(helper, "version", "-o", "json")},
		{"oci-bassh-version.schema.json", append(helper, "--version", "--json")},
		{"oci-bassh-contract-check.schema.json", append(helper, "contract-check")},
	}

	for _, check := range checks {
		t.Run(strings.Join(check.args[1:], "_"), func(t *testing.T) {
			run := runCommandForTest(t, check.args, env)
			if run.code != 0 {
				t.Fatalf("command failed with %d\nstdout:\n%s\nstderr:\n%s", run.code, run.stdout, run.stderr)
			}
			payload := decodeObject(t, run.stdout)
			assertRequiredKeys(t, check.schema, payload)
		})
	}

	failEnv := append([]string{}, env...)
	failEnv = append(failEnv, "BASTION_SESSION_FAIL_DOCTOR=1")
	doctor := runCommandForTest(t, append(helper, "doctor", "my-vps-01"), failEnv)
	if doctor.code != 0 {
		t.Fatalf("doctor should report unhealthy diagnostics without failing, got %d\nstdout:\n%s\nstderr:\n%s", doctor.code, doctor.stdout, doctor.stderr)
	}
	doctorPayload := decodeObject(t, doctor.stdout)
	assertRequiredKeys(t, "oci-bassh-doctor.schema.json", doctorPayload)
	if doctorPayload["ok"] != false || doctorPayload["issue"] == nil {
		t.Fatalf("doctor failure payload should include ok=false and issue, got %#v", doctorPayload)
	}
	check := runCommandForTest(t, append(helper, "check", "my-vps-01"), failEnv)
	if check.code == 0 {
		t.Fatalf("check should fail when diagnostics are unhealthy\nstdout:\n%s", check.stdout)
	}
	checkPayload := decodeObject(t, check.stdout)
	assertRequiredKeys(t, "oci-bassh-check.schema.json", checkPayload)
}

func buildOCIBassh(t *testing.T, tmp string) string {
	t.Helper()
	out := filepath.Join(tmp, "oci-bassh")
	cmd := exec.Command("go", "build", "-o", out, "./cmd/oci-bassh")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(output))
	}
	return out
}

func writeHermeticShims(t *testing.T, binDir string) {
	t.Helper()
	writeExecutable(t, filepath.Join(binDir, "oci-context"), `#!/bin/sh
set -eu
case "$*" in
  "auth ensure --output json")
    printf '{"ok":true,"state":"ready","context":"dev","profile":"DEFAULT","auth_method":"api_key"}\n'
    ;;
  "status --cached -o json")
    printf '{"context":"dev","profile":"DEFAULT","region":"us-phoenix-1","auth_method":"api_key"}\n'
    ;;
  "doctor -o json")
    printf '{"auth_ensure":{"ok":true,"state":"ready"},"current_context":"dev"}\n'
    ;;
  "auth show --output json")
    printf '{"context":"dev","daemon_available":false}\n'
    ;;
  "--version"|"-v"|"version")
    printf '0.99.0\n'
    ;;
  *)
    printf 'unexpected oci-context %s\n' "$*" >&2
    exit 2
    ;;
esac
`)
	writeExecutable(t, filepath.Join(binDir, "bastion-session"), `#!/bin/sh
set -eu
host="${2:-my-vps-01}"
case "$*" in
  "--version"|"-v"|"version")
    printf '0.99.0\n'
    ;;
  "target list -o json")
    printf '[{"name":"my-vps-01"}]\n'
    ;;
  "doctor -o json"|"doctor my-vps-01 -o json"|"doctor my-vps-01 --cached -o json"|"doctor my-vps-01 --fix -o json")
    if [ "${BASTION_SESSION_FAIL_DOCTOR:-}" ]; then
      printf '{"ok":false,"issues":[{"message":"broken fixture"}]}\n'
      printf 'broken fixture\n' >&2
      exit 1
    fi
    printf '{"current_bastion":{"available":true},"session":{"cached":{"lifecycle":"ACTIVE"}},"ssh_include":{"exists":true}}\n'
    ;;
  "target import my-vps-01 --terraform-outputs "*)
    printf '{"tracked":true,"host":"my-vps-01"}\n'
    ;;
  "target show my-vps-01 -o json")
    printf '{"name":"my-vps-01","instance_id":"ocid1.instance","private_ip":"10.0.1.25"}\n'
    ;;
  "ensure my-vps-01 -o json")
    printf '{"ready":true,"ssh_host":"my-vps-01","connect_command":"ssh my-vps-01","target_private_ip":"10.0.1.25"}\n'
    ;;
  "explain my-vps-01 -o json")
    printf '{"host":"my-vps-01","target":"10.0.1.25","proxyjump":"my-bastion","connect_command":"ssh my-vps-01"}\n'
    ;;
  "ssh-config show my-vps-01 -o json")
    printf '{"host":"my-vps-01","hostname":"10.0.1.25","user":"cloud-user","proxyjump":"my-bastion"}\n'
    ;;
  status*)
    printf '{"session_id":"ocid1.session","lifecycle":"ACTIVE"}\n'
    ;;
  *)
    printf 'unexpected bastion-session %s\n' "$*" >&2
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

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func runCommandForTest(t *testing.T, args []string, env []string) commandRun {
	t.Helper()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = env
	stdout, err := cmd.Output()
	if err == nil {
		return commandRun{stdout: string(stdout)}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return commandRun{stdout: string(stdout), stderr: string(exitErr.Stderr), code: exitErr.ExitCode()}
	}
	t.Fatalf("command failed before exit: %v", err)
	return commandRun{}
}

func decodeObject(t *testing.T, raw string) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("output is not a JSON object: %v\n%s", err, raw)
	}
	return payload
}

func assertRequiredKeys(t *testing.T, schema string, payload map[string]any) {
	t.Helper()
	for _, key := range requiredKeys[schema] {
		if _, ok := payload[key]; !ok {
			t.Fatalf("%s: missing required key %q in %#v", schema, key, payload)
		}
	}
	if ok, exists := payload["ok"]; exists {
		if _, isBool := ok.(bool); !isBool {
			t.Fatalf("%s: ok must be boolean, got %T", schema, ok)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func mustGlob(t *testing.T, pattern string) []string {
	t.Helper()
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatalf("no matches for %s", pattern)
	}
	return matches
}
