#!/usr/bin/env python3
import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path


def run_json(argv, required=False):
    proc = subprocess.run(argv, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    payload = None
    if proc.stdout.strip():
        try:
            payload = json.loads(proc.stdout)
        except json.JSONDecodeError:
            payload = {"raw_stdout": proc.stdout}
    result = {
        "command": argv,
        "ok": proc.returncode == 0,
        "exit_code": proc.returncode,
        "stdout": proc.stdout,
        "stderr": proc.stderr,
        "json": payload,
    }
    if required and proc.returncode != 0:
        raise RuntimeError(json.dumps(result, indent=2))
    return result


def emit(obj, code=0):
    print(json.dumps(obj, indent=2, sort_keys=True))
    return code


def which(name):
    return shutil.which(name)


def doctor(_args):
    tools = {name: which(name) for name in ["oci-context", "bastion-session", "ssh"]}
    context = run_json(["oci-context", "status", "--cached", "-o", "json"]) if tools["oci-context"] else None
    auth = run_json(["oci-context", "auth", "ensure", "--output", "json"]) if tools["oci-context"] else None
    bastion = run_json(["bastion-session", "status", "-o", "json"]) if tools["bastion-session"] else None
    targets = run_json(["bastion-session", "target", "list", "-o", "json"]) if tools["bastion-session"] else None
    ok = bool(tools["oci-context"] and tools["bastion-session"] and tools["ssh"])
    if auth is not None:
        ok = ok and auth["ok"] and bool((auth.get("json") or {}).get("ok", auth["ok"]))
    return emit({
        "ok": ok,
        "tools": tools,
        "context": context,
        "auth": auth,
        "bastion_status": bastion,
        "targets": targets,
    }, 0 if ok else 1)


def ensure_target(args):
    auth = run_json(["oci-context", "auth", "ensure", "--output", "json"])
    ensure_cmd = ["bastion-session", "ensure", args.host, "-o", "json"]
    if args.identity_file:
        ensure_cmd += ["--identity-file", args.identity_file]
    ensured = run_json(ensure_cmd)
    ssh_config = run_json(["bastion-session", "ssh-config", "show", args.host, "-o", "json"])
    ok = auth["ok"] and ensured["ok"] and ssh_config["ok"]
    return emit({
        "ok": ok,
        "host": args.host,
        "auth": auth,
        "ensure": ensured,
        "ssh_config": ssh_config,
        "connect_command": ((ensured.get("json") or {}).get("connect_command") or f"ssh {args.host}"),
    }, 0 if ok else 1)


def track_from_terraform(args):
    cmd = [
        "bastion-session", "target", "import", args.host,
        "--terraform-outputs", args.terraform_outputs,
    ]
    if args.user:
        cmd += ["--user", args.user]
    if args.identity_file:
        cmd += ["--identity-file", args.identity_file]
    tracked = run_json(cmd)
    shown = run_json(["bastion-session", "target", "show", args.host, "-o", "json"])
    ok = tracked["ok"] and shown["ok"]
    return emit({"ok": ok, "host": args.host, "track": tracked, "target": shown}, 0 if ok else 1)


def contract_check(_args):
    checks = []
    commands = [
        ["oci-context", "auth", "ensure", "--output", "json"],
        ["oci-context", "status", "--cached", "-o", "json"],
        ["bastion-session", "target", "list", "-o", "json"],
    ]
    for cmd in commands:
        checks.append(run_json(cmd))
    ok = all(item["ok"] and isinstance(item.get("json"), (dict, list)) for item in checks)
    return emit({"ok": ok, "checks": checks}, 0 if ok else 1)


def main():
    parser = argparse.ArgumentParser(description="OCI remote workstation helper")
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("doctor")
    p.set_defaults(func=doctor)

    p = sub.add_parser("ensure-target")
    p.add_argument("host")
    p.add_argument("--identity-file")
    p.set_defaults(func=ensure_target)

    p = sub.add_parser("track-from-terraform")
    p.add_argument("host")
    p.add_argument("terraform_outputs")
    p.add_argument("--user")
    p.add_argument("--identity-file")
    p.set_defaults(func=track_from_terraform)

    p = sub.add_parser("contract-check")
    p.set_defaults(func=contract_check)

    args = parser.parse_args()
    try:
        return args.func(args)
    except RuntimeError as exc:
        print(str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

