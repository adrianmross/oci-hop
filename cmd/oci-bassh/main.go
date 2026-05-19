package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type commandResult struct {
	Command  []string         `json:"command"`
	OK       bool             `json:"ok"`
	ExitCode int              `json:"exit_code"`
	Stdout   string           `json:"stdout"`
	Stderr   string           `json:"stderr"`
	JSON     *json.RawMessage `json:"json,omitempty"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage(os.Stdout)
		return nil
	case "-v", "--version", "version":
		fmt.Println(version)
		return nil
	case "-vv":
		fmt.Printf("%s (commit=%s date=%s)\n", version, commit, date)
		return nil
	case "doctor":
		return cmdDoctor(args[1:])
	case "ensure", "ensure-target":
		return cmdEnsure(args[1:])
	case "track", "track-from-terraform":
		return cmdTrack(args[1:])
	case "ssh":
		return cmdSSH(args[1:])
	case "contract-check":
		return cmdContractCheck(args[1:])
	default:
		return cliError{code: 2, msg: "unknown command: " + args[0]}
	}
}

func usage(w *os.File) {
	fmt.Fprintln(w, `oci-bassh manages SSH to OCI compute hosts through OCI Bastion.

Usage:
  oci-bassh doctor [host]
  oci-bassh track <host> <terraform-outputs> [--user USER] [--identity-file PATH]
  oci-bassh ensure <host> [--identity-file PATH]
  oci-bassh ssh [--dry-run] [--identity-file PATH] <host> [-- ssh args...]
  oci-bassh contract-check`)
}

func cmdDoctor(args []string) error {
	host, err := optionalHost(args)
	if err != nil {
		return err
	}
	tools := map[string]string{}
	for _, name := range []string{"oci-context", "bastion-session", "ssh"} {
		if p, err := exec.LookPath(name); err == nil {
			tools[name] = p
		}
	}
	oci := runJSON("oci-context", "doctor", "-o", "json")
	var bastion commandResult
	if host == "" {
		bastion = runJSON("bastion-session", "doctor", "-o", "json")
	} else {
		bastion = runJSON("bastion-session", "doctor", host, "-o", "json")
	}
	targets := runJSON("bastion-session", "target", "list", "-o", "json")
	ok := oci.OK && bastion.OK && targets.OK && tools["oci-context"] != "" && tools["bastion-session"] != "" && tools["ssh"] != ""
	out := map[string]any{
		"ok":             ok,
		"host":           host,
		"tools":          tools,
		"oci_context":    oci,
		"bastion_doctor": bastion,
		"targets":        targets,
	}
	if err := emit(out); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "doctor found issues"}
	}
	return nil
}

func cmdEnsure(args []string) error {
	host, identityFile, err := parseHostIdentity(args)
	if err != nil {
		return err
	}
	auth := runJSON("oci-context", "auth", "ensure", "--output", "json")
	ensureArgs := []string{"ensure", host, "-o", "json"}
	if identityFile != "" {
		ensureArgs = append(ensureArgs, "--identity-file", identityFile)
	}
	ensured := runJSON("bastion-session", ensureArgs...)
	sshConfig := runJSON("bastion-session", "ssh-config", "show", host, "-o", "json")
	ok := auth.OK && ensured.OK && sshConfig.OK
	connectCommand := "ssh " + host
	if ensured.JSON != nil {
		var obj map[string]any
		if json.Unmarshal(*ensured.JSON, &obj) == nil {
			if v, _ := obj["connect_command"].(string); v != "" {
				connectCommand = v
			}
		}
	}
	if err := emit(map[string]any{
		"ok":              ok,
		"host":            host,
		"auth":            auth,
		"ensure":          ensured,
		"ssh_config":      sshConfig,
		"connect_command": connectCommand,
	}); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "ensure failed"}
	}
	return nil
}

func cmdTrack(args []string) error {
	if len(args) < 2 {
		return cliError{code: 2, msg: "track requires <host> <terraform-outputs>"}
	}
	host := args[0]
	tf := args[1]
	cmdArgs := []string{"target", "import", host, "--terraform-outputs", tf}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--user", "--identity-file":
			if i+1 >= len(args) {
				return cliError{code: 2, msg: args[i] + " requires a value"}
			}
			cmdArgs = append(cmdArgs, args[i], args[i+1])
			i++
		default:
			return cliError{code: 2, msg: "unknown track flag: " + args[i]}
		}
	}
	tracked := runJSON("bastion-session", cmdArgs...)
	shown := runJSON("bastion-session", "target", "show", host, "-o", "json")
	ok := tracked.OK && shown.OK
	if err := emit(map[string]any{"ok": ok, "host": host, "track": tracked, "target": shown}); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "track failed"}
	}
	return nil
}

func cmdSSH(args []string) error {
	host, identityFile, dryRun, sshArgs, err := parseSSHArgs(args)
	if err != nil {
		return err
	}
	auth := runJSON("oci-context", "auth", "ensure", "--output", "json")
	ensureArgs := []string{"ensure", host, "-o", "json"}
	if identityFile != "" {
		ensureArgs = append(ensureArgs, "--identity-file", identityFile)
	}
	ensured := runJSON("bastion-session", ensureArgs...)
	ok := auth.OK && ensured.OK
	sshCmd := append([]string{"ssh", host}, sshArgs...)
	if dryRun {
		if err := emit(map[string]any{"ok": ok, "host": host, "auth": auth, "ensure": ensured, "ssh_command": sshCmd}); err != nil {
			return err
		}
		if !ok {
			return cliError{code: 1, msg: "ssh preparation failed"}
		}
		return nil
	}
	if !ok {
		if err := emit(map[string]any{"ok": false, "host": host, "auth": auth, "ensure": ensured, "ssh_command": sshCmd}); err != nil {
			return err
		}
		return cliError{code: 1, msg: "ssh preparation failed"}
	}
	return syscallExec("ssh", sshCmd)
}

func cmdContractCheck(args []string) error {
	if len(args) != 0 {
		return cliError{code: 2, msg: "contract-check accepts no arguments"}
	}
	checks := []commandResult{
		runJSON("oci-context", "auth", "ensure", "--output", "json"),
		runJSON("oci-context", "status", "--cached", "-o", "json"),
		runJSON("bastion-session", "target", "list", "-o", "json"),
	}
	ok := true
	for _, check := range checks {
		ok = ok && check.OK && check.JSON != nil
	}
	if err := emit(map[string]any{"ok": ok, "checks": checks}); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "contract check failed"}
	}
	return nil
}

func runJSON(name string, args ...string) commandResult {
	cmdArgs := append([]string{name}, args...)
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	rc := 0
	if err != nil {
		rc = 1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			rc = ee.ExitCode()
		}
	}
	var raw *json.RawMessage
	trimmed := bytes.TrimSpace(stdout.Bytes())
	if len(trimmed) > 0 && json.Valid(trimmed) {
		cp := json.RawMessage(append([]byte(nil), trimmed...))
		raw = &cp
	}
	return commandResult{Command: cmdArgs, OK: err == nil, ExitCode: rc, Stdout: stdout.String(), Stderr: stderr.String(), JSON: raw}
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func optionalHost(args []string) (string, error) {
	if len(args) > 1 {
		return "", cliError{code: 2, msg: "doctor accepts at most one host"}
	}
	if len(args) == 0 {
		return "", nil
	}
	return strings.TrimSpace(args[0]), nil
}

func parseHostIdentity(args []string) (host, identityFile string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--identity-file":
			if i+1 >= len(args) {
				return "", "", cliError{code: 2, msg: "--identity-file requires a value"}
			}
			identityFile = args[i+1]
			i++
		default:
			if host != "" {
				return "", "", cliError{code: 2, msg: "unexpected argument: " + args[i]}
			}
			host = args[i]
		}
	}
	if strings.TrimSpace(host) == "" {
		return "", "", cliError{code: 2, msg: "host is required"}
	}
	return host, identityFile, nil
}

func parseSSHArgs(args []string) (host, identityFile string, dryRun bool, sshArgs []string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--":
			sshArgs = append(sshArgs, args[i+1:]...)
			return requireHost(host, identityFile, dryRun, sshArgs)
		case "--dry-run":
			dryRun = true
		case "--identity-file":
			if i+1 >= len(args) {
				return "", "", false, nil, cliError{code: 2, msg: "--identity-file requires a value"}
			}
			identityFile = args[i+1]
			i++
		default:
			if host != "" {
				sshArgs = append(sshArgs, args[i:]...)
				return requireHost(host, identityFile, dryRun, sshArgs)
			}
			host = args[i]
		}
	}
	return requireHost(host, identityFile, dryRun, sshArgs)
}

func requireHost(host, identityFile string, dryRun bool, sshArgs []string) (string, string, bool, []string, error) {
	if strings.TrimSpace(host) == "" {
		return "", "", false, nil, cliError{code: 2, msg: "host is required"}
	}
	return host, identityFile, dryRun, sshArgs, nil
}

type cliError struct {
	code int
	msg  string
}

func (e cliError) Error() string { return e.msg }

func exitCode(err error) int {
	var ce cliError
	if errors.As(err, &ce) && ce.code > 0 {
		return ce.code
	}
	return 1
}

func syscallExec(name string, argv []string) error {
	cmd := exec.Command(name, argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
