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
	Command     []string         `json:"command"`
	OK          bool             `json:"ok"`
	ExitCode    int              `json:"exit_code"`
	Stdout      string           `json:"stdout"`
	Stderr      string           `json:"stderr"`
	ErrorCode   string           `json:"error_code,omitempty"`
	Message     string           `json:"message,omitempty"`
	NextCommand string           `json:"next_command,omitempty"`
	JSON        *json.RawMessage `json:"json,omitempty"`
}

type issue struct {
	ErrorCode   string `json:"error_code"`
	Message     string `json:"message"`
	NextCommand string `json:"next_command,omitempty"`
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
	case "check":
		return cmdCheck(args[1:])
	case "inspect":
		return cmdInspect(args[1:])
	case "repair":
		return cmdRepair(args[1:])
	case "ensure", "ensure-target":
		return cmdEnsure(args[1:])
	case "track", "track-from-terraform":
		return cmdTrack(args[1:])
	case "ssh":
		return cmdSSH(args[1:])
	case "completion":
		return cmdCompletion(args[1:])
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
  oci-bassh check [host]
  oci-bassh inspect <host>
  oci-bassh repair <host> [--ensure] [--identity-file PATH]
  oci-bassh track <host> <terraform-outputs> [--user USER] [--identity-file PATH]
  oci-bassh track <host> --terraform-dir DIR [--user USER] [--identity-file PATH]
  oci-bassh ensure <host> [--identity-file PATH]
  oci-bassh ssh [--dry-run] [--identity-file PATH] <host> [-- ssh args...]
  oci-bassh completion bash|zsh|fish
  oci-bassh contract-check`)
}

func cmdDoctor(args []string) error {
	host, err := optionalHost(args)
	if err != nil {
		return err
	}
	out := doctorPayload(host)
	return emit(out)
}

func cmdCheck(args []string) error {
	host, err := optionalHostFor("check", args)
	if err != nil {
		return err
	}
	out := doctorPayload(host)
	if err := emit(out); err != nil {
		return err
	}
	if ok, _ := out["ok"].(bool); !ok {
		return cliError{code: 1, msg: "check found issues"}
	}
	return nil
}

func doctorPayload(host string) map[string]any {
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
		"versions":       versions(),
		"oci_context":    oci,
		"bastion_doctor": bastion,
		"targets":        targets,
	}
	if !ok {
		out["issue"] = firstIssue("doctor found issues", nextForHost("oci-bassh repair", host), oci, bastion, targets)
	}
	return out
}

func cmdInspect(args []string) error {
	host, err := requiredHostOnly("inspect", args)
	if err != nil {
		return err
	}
	status := runJSON("oci-context", "status", "--cached", "-o", "json")
	auth := runJSON("oci-context", "auth", "show", "--output", "json")
	bastion := runJSON("bastion-session", "doctor", host, "--cached", "-o", "json")
	sshConfig := runJSON("bastion-session", "ssh-config", "show", host, "-o", "json")
	sshEffective := runCommand("ssh", "-G", host)
	ok := status.OK && auth.OK && bastion.OK && sshConfig.OK && sshEffective.OK
	out := map[string]any{
		"ok":             ok,
		"host":           host,
		"versions":       versions(),
		"oci_status":     status,
		"auth":           auth,
		"bastion_doctor": bastion,
		"ssh_config":     sshConfig,
		"ssh_effective":  sshEffective,
	}
	if !ok {
		out["issue"] = firstIssue("inspect found issues", nextForHost("oci-bassh repair", host), status, auth, bastion, sshConfig, sshEffective)
	}
	return emit(out)
}

func cmdRepair(args []string) error {
	host, identityFile, ensure, err := parseRepairArgs(args)
	if err != nil {
		return err
	}
	repaired := runJSON("bastion-session", "doctor", host, "--fix", "-o", "json")
	var auth commandResult
	var ensured commandResult
	var sshConfig commandResult
	connectCommand := "ssh " + host
	ok := repaired.OK
	if ensure {
		auth = runJSON("oci-context", "auth", "ensure", "--output", "json")
		ensureArgs := []string{"ensure", host, "-o", "json"}
		if identityFile != "" {
			ensureArgs = append(ensureArgs, "--identity-file", identityFile)
		}
		ensured = runJSON("bastion-session", ensureArgs...)
		sshConfig = runJSON("bastion-session", "ssh-config", "show", host, "-o", "json")
		ok = auth.OK && ensured.OK && sshConfig.OK
		connectCommand = connectCommandFrom(host, ensured)
	}
	out := map[string]any{
		"ok":               ok,
		"host":             host,
		"repair":           repaired,
		"ensure_requested": ensure,
		"connect_command":  connectCommand,
	}
	if ensure {
		out["auth"] = auth
		out["ensure"] = ensured
		out["ssh_config"] = sshConfig
		if !repaired.OK {
			out["repair_issue"] = firstIssue("repair found remaining issues", nextForHost("oci-bassh inspect", host), repaired)
		}
	}
	if !ok {
		out["issue"] = firstIssue("repair failed", nextForHost("oci-bassh inspect", host), auth, ensured, sshConfig, repaired)
	}
	if err := emit(out); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "repair failed"}
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
	connectCommand := connectCommandFrom(host, ensured)
	out := map[string]any{
		"ok":              ok,
		"host":            host,
		"auth":            auth,
		"ensure":          ensured,
		"ssh_config":      sshConfig,
		"connect_command": connectCommand,
	}
	if !ok {
		out["issue"] = firstIssue("ensure failed", nextForHost("oci-bassh repair --ensure", host), auth, ensured, sshConfig)
	}
	if err := emit(out); err != nil {
		return err
	}
	if !ok {
		return cliError{code: 1, msg: "ensure failed"}
	}
	return nil
}

func cmdTrack(args []string) error {
	host, tf, passthrough, err := parseTrackArgs(args)
	if err != nil {
		return err
	}
	cmdArgs := []string{"target", "import", host, "--terraform-outputs", tf}
	cmdArgs = append(cmdArgs, passthrough...)
	tracked := runJSON("bastion-session", cmdArgs...)
	shown := runJSON("bastion-session", "target", "show", host, "-o", "json")
	ok := tracked.OK && shown.OK
	out := map[string]any{"ok": ok, "host": host, "track": tracked, "target": shown}
	if !ok {
		out["issue"] = firstIssue("track failed", nextForHost("oci-bassh inspect", host), tracked, shown)
	}
	if err := emit(out); err != nil {
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
		out := map[string]any{"ok": ok, "host": host, "auth": auth, "ensure": ensured, "ssh_command": sshCmd}
		if !ok {
			out["issue"] = firstIssue("ssh preparation failed", nextForHost("oci-bassh repair --ensure", host), auth, ensured)
		}
		if err := emit(out); err != nil {
			return err
		}
		if !ok {
			return cliError{code: 1, msg: "ssh preparation failed"}
		}
		return nil
	}
	if !ok {
		out := map[string]any{"ok": false, "host": host, "auth": auth, "ensure": ensured, "ssh_command": sshCmd, "issue": firstIssue("ssh preparation failed", nextForHost("oci-bassh repair --ensure", host), auth, ensured)}
		if err := emit(out); err != nil {
			return err
		}
		return cliError{code: 1, msg: "ssh preparation failed"}
	}
	return syscallExec("ssh", sshCmd)
}

func cmdCompletion(args []string) error {
	if len(args) != 1 {
		return cliError{code: 2, msg: "completion requires bash, zsh, or fish"}
	}
	switch args[0] {
	case "bash":
		fmt.Print(`_oci_bassh_complete()
{
  local cur="${COMP_WORDS[COMP_CWORD]}"
  local cmds="doctor check inspect repair track track-from-terraform ensure ensure-target ssh completion contract-check version"
  COMPREPLY=( $(compgen -W "$cmds" -- "$cur") )
}
complete -F _oci_bassh_complete oci-bassh
`)
	case "zsh":
		fmt.Print(`#compdef oci-bassh
_arguments '1:command:(doctor check inspect repair track track-from-terraform ensure ensure-target ssh completion contract-check version)'
`)
	case "fish":
		fmt.Print(`complete -c oci-bassh -f -a "doctor check inspect repair track track-from-terraform ensure ensure-target ssh completion contract-check version"
`)
	default:
		return cliError{code: 2, msg: "completion requires bash, zsh, or fish"}
	}
	return nil
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
	return runCommand(name, args...)
}

func runCommand(name string, args ...string) commandResult {
	cmdArgs := append([]string{name}, args...)
	cmd := exec.Command(name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	rc := 0
	errorCode := ""
	message := ""
	if err != nil {
		rc = 1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			rc = ee.ExitCode()
			errorCode = "command_failed"
			message = strings.TrimSpace(stderr.String())
			if message == "" {
				message = err.Error()
			}
		} else {
			var execErr *exec.Error
			if errors.As(err, &execErr) {
				rc = 127
				errorCode = "command_not_found"
				message = execErr.Error()
			} else {
				errorCode = "command_error"
				message = err.Error()
			}
		}
	}
	var raw *json.RawMessage
	trimmed := bytes.TrimSpace(stdout.Bytes())
	if len(trimmed) > 0 && json.Valid(trimmed) {
		cp := json.RawMessage(append([]byte(nil), trimmed...))
		raw = &cp
	}
	return commandResult{Command: cmdArgs, OK: err == nil, ExitCode: rc, Stdout: stdout.String(), Stderr: stderr.String(), ErrorCode: errorCode, Message: message, JSON: raw}
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func optionalHost(args []string) (string, error) {
	return optionalHostFor("doctor", args)
}

func optionalHostFor(command string, args []string) (string, error) {
	if len(args) > 1 {
		return "", cliError{code: 2, msg: command + " accepts at most one host"}
	}
	if len(args) == 0 {
		return "", nil
	}
	return strings.TrimSpace(args[0]), nil
}

func requiredHostOnly(command string, args []string) (string, error) {
	if len(args) != 1 {
		return "", cliError{code: 2, msg: command + " requires exactly one host"}
	}
	host := strings.TrimSpace(args[0])
	if host == "" {
		return "", cliError{code: 2, msg: "host is required"}
	}
	return host, nil
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

func parseRepairArgs(args []string) (host, identityFile string, ensure bool, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--ensure":
			ensure = true
		case "--identity-file":
			if i+1 >= len(args) {
				return "", "", false, cliError{code: 2, msg: "--identity-file requires a value"}
			}
			identityFile = args[i+1]
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", false, cliError{code: 2, msg: "unknown repair flag: " + args[i]}
			}
			if host != "" {
				return "", "", false, cliError{code: 2, msg: "unexpected argument: " + args[i]}
			}
			host = args[i]
		}
	}
	if strings.TrimSpace(host) == "" {
		return "", "", false, cliError{code: 2, msg: "host is required"}
	}
	return host, identityFile, ensure, nil
}

func parseTrackArgs(args []string) (host, terraformOutputs string, passthrough []string, err error) {
	if len(args) < 1 {
		return "", "", nil, cliError{code: 2, msg: "track requires <host> <terraform-outputs> or <host> --terraform-dir DIR"}
	}
	host = strings.TrimSpace(args[0])
	if host == "" {
		return "", "", nil, cliError{code: 2, msg: "host is required"}
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--terraform-dir":
			if i+1 >= len(args) {
				return "", "", nil, cliError{code: 2, msg: "--terraform-dir requires a value"}
			}
			if terraformOutputs != "" {
				return "", "", nil, cliError{code: 2, msg: "terraform outputs specified more than once"}
			}
			terraformOutputs = args[i+1]
			i++
		case "--user", "--identity-file":
			if i+1 >= len(args) {
				return "", "", nil, cliError{code: 2, msg: args[i] + " requires a value"}
			}
			passthrough = append(passthrough, args[i], args[i+1])
			i++
		default:
			if strings.HasPrefix(args[i], "-") {
				return "", "", nil, cliError{code: 2, msg: "unknown track flag: " + args[i]}
			}
			if terraformOutputs != "" {
				return "", "", nil, cliError{code: 2, msg: "unexpected argument: " + args[i]}
			}
			terraformOutputs = args[i]
		}
	}
	if strings.TrimSpace(terraformOutputs) == "" {
		return "", "", nil, cliError{code: 2, msg: "terraform outputs path is required"}
	}
	return host, terraformOutputs, passthrough, nil
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

func connectCommandFrom(host string, result commandResult) string {
	if result.JSON != nil {
		var obj map[string]any
		if json.Unmarshal(*result.JSON, &obj) == nil {
			if v, _ := obj["connect_command"].(string); v != "" {
				return v
			}
		}
	}
	return "ssh " + host
}

func versions() map[string]commandResult {
	return map[string]commandResult{
		"oci_bassh":       {Command: []string{"oci-bassh", "--version"}, OK: true, ExitCode: 0, Stdout: version + "\n"},
		"oci_context":     runCommand("oci-context", "--version"),
		"bastion_session": runCommand("bastion-session", "--version"),
		"ssh":             runCommand("ssh", "-V"),
	}
}

func firstIssue(message, nextCommand string, results ...commandResult) issue {
	for _, result := range results {
		if len(result.Command) == 0 || result.OK {
			continue
		}
		code := result.ErrorCode
		if code == "" {
			code = "command_failed"
		}
		msg := result.Message
		if msg == "" {
			msg = strings.TrimSpace(result.Stderr)
		}
		if msg == "" {
			msg = message
		}
		return issue{ErrorCode: code, Message: msg, NextCommand: nextCommand}
	}
	return issue{ErrorCode: "not_ok", Message: message, NextCommand: nextCommand}
}

func nextForHost(prefix, host string) string {
	if strings.TrimSpace(host) == "" {
		return strings.TrimSpace(prefix)
	}
	return strings.TrimSpace(prefix) + " " + host
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
