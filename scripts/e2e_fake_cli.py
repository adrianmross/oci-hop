#!/usr/bin/env python3
import json
import os
import stat
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def write_exe(path, text):
    path.write_text(text)
    path.chmod(path.stat().st_mode | stat.S_IXUSR)


def run(cmd, env):
    return subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)


REQUIRED = {
    "oci-bassh-doctor.schema.json": ["ok", "tools", "oci_context", "bastion_doctor", "targets"],
    "oci-bassh-ensure.schema.json": ["ok", "host", "auth", "ensure", "ssh_config", "connect_command"],
    "oci-bassh-track.schema.json": ["ok", "host", "track", "target"],
    "oci-bassh-ssh.schema.json": ["ok", "host", "auth", "ensure", "ssh_command"],
    "oci-bassh-contract-check.schema.json": ["ok", "checks"],
}


def validate(schema, data):
    for key in REQUIRED[schema]:
        if key not in data:
            raise AssertionError(f"{schema}: missing {key}")


def helper_cmd(tmp):
    configured = os.environ.get("OCI_BASSH_BIN")
    if configured:
        return [configured]
    binary = tmp / "oci-bassh"
    proc = subprocess.run(["go", "build", "-o", str(binary), "./cmd/oci-bassh"], cwd=ROOT, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    if proc.returncode != 0:
        print(proc.stdout)
        print(proc.stderr, file=sys.stderr)
        raise SystemExit(proc.returncode)
    return [str(binary)]


def main():
    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        bin_dir = tmp / "bin"
        bin_dir.mkdir()
        state = tmp / "state.json"
        state.write_text("{}")

        write_exe(bin_dir / "oci-context", f"""#!/usr/bin/env python3
import json, sys
args = sys.argv[1:]
if args[:2] == ["auth", "ensure"]:
    print(json.dumps({{"ok": True, "state": "ready", "context": "dev", "profile": "DEFAULT", "auth_method": "api_key"}}))
elif args[:1] == ["status"]:
    print(json.dumps({{"context": "dev", "profile": "DEFAULT", "region": "us-phoenix-1", "auth_method": "api_key"}}))
elif args[:1] == ["doctor"]:
    print(json.dumps({{"auth_ensure": {{"ok": True, "state": "ready"}}, "current_context": "dev"}}))
elif args[:2] == ["auth", "show"]:
    print(json.dumps({{"context": "dev", "daemon_available": False}}))
else:
    print("unexpected oci-context " + " ".join(args), file=sys.stderr)
    sys.exit(2)
""")

        write_exe(bin_dir / "bastion-session", f"""#!/usr/bin/env python3
import json, sys
args = sys.argv[1:]
if args[:3] == ["target", "list", "-o"]:
    print(json.dumps([]))
elif args[:1] == ["doctor"]:
    print(json.dumps({{"current_bastion": {{"available": True}}, "session": {{"cached": {{"lifecycle": "ACTIVE"}}}}, "ssh_include": {{"exists": True}}}}))
elif args[:2] == ["target", "import"]:
    print("Tracked target " + args[2])
elif args[:2] == ["target", "show"]:
    print(json.dumps({{"name": args[2], "instance_id": "ocid1.instance", "private_ip": "10.0.0.5"}}))
elif args[:1] == ["ensure"]:
    print(json.dumps({{"ready": True, "ssh_host": args[1], "connect_command": "ssh " + args[1], "target_private_ip": "10.0.0.5"}}))
elif len(args) >= 3 and args[:2] == ["ssh-config", "show"]:
    print(json.dumps({{"host": args[2], "hostname": "10.0.0.5", "user": "opc", "proxyjump": "DEFAULT-bastion"}}))
elif args[:1] == ["status"]:
    print(json.dumps({{"session_id": "ocid1.session", "lifecycle": "ACTIVE"}}))
else:
    print("unexpected bastion-session " + " ".join(args), file=sys.stderr)
    sys.exit(2)
""")

        write_exe(bin_dir / "ssh", """#!/bin/sh
if [ "$1" = "-G" ]; then
  echo "user opc"
  echo "hostname 10.0.0.5"
  echo "proxyjump DEFAULT-bastion"
  exit 0
fi
exit 2
""")

        env = os.environ.copy()
        env["PATH"] = str(bin_dir) + os.pathsep + env["PATH"]
        helper = helper_cmd(tmp)

        checks = [
            ("oci-bassh-doctor.schema.json", helper + ["doctor"]),
            ("oci-bassh-ensure.schema.json", helper + ["ensure-target", "vmordws02"]),
            ("oci-bassh-ensure.schema.json", helper + ["ensure", "vmordws02"]),
            ("oci-bassh-track.schema.json", helper + ["track-from-terraform", "vmordws02", str(tmp)]),
            ("oci-bassh-track.schema.json", helper + ["track", "vmordws02", str(tmp)]),
            ("oci-bassh-ssh.schema.json", helper + ["ssh", "--dry-run", "vmordws02"]),
            ("oci-bassh-contract-check.schema.json", helper + ["contract-check"]),
        ]
        for schema, cmd in checks:
            proc = run(cmd, env)
            if proc.returncode != 0:
                print(proc.stdout)
                print(proc.stderr, file=sys.stderr)
                return proc.returncode
            validate(schema, json.loads(proc.stdout))
        print("e2e fake CLI passed")
        return 0


if __name__ == "__main__":
    raise SystemExit(main())
