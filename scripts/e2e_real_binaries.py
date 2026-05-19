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


def copy_or_link(src, dst):
    if src:
        os.symlink(Path(src).resolve(), dst)


def run(cmd, env):
    proc = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    if proc.returncode != 0:
        print(proc.stdout)
        print(proc.stderr, file=sys.stderr)
    return proc


REQUIRED = {
    "oci-bassh-doctor.schema.json": ["ok", "tools", "versions", "oci_context", "bastion_doctor", "targets"],
    "oci-bassh-check.schema.json": ["ok", "tools", "versions", "oci_context", "bastion_doctor", "targets"],
    "oci-bassh-inspect.schema.json": ["ok", "host", "versions", "oci_status", "auth", "bastion_doctor", "ssh_config", "ssh_effective"],
    "oci-bassh-repair.schema.json": ["ok", "host", "repair", "ensure_requested", "connect_command"],
    "oci-bassh-ensure.schema.json": ["ok", "host", "auth", "ensure", "ssh_config", "connect_command"],
    "oci-bassh-track.schema.json": ["ok", "host", "track", "target"],
    "oci-bassh-ssh.schema.json": ["ok", "host", "auth", "ensure", "ssh_command"],
    "oci-bassh-paths.schema.json": ["ok", "paths"],
    "oci-bassh-upgrade.schema.json": ["ok", "dry_run", "command"],
    "oci-bassh-version.schema.json": ["ok", "version", "commit", "date"],
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
    oci_context = os.environ.get("OCI_CONTEXT_BIN", "oci-context")
    bastion_session = os.environ.get("BASTION_SESSION_BIN", "bastion-session")

    with tempfile.TemporaryDirectory() as td:
        tmp = Path(td)
        bin_dir = tmp / "bin"
        home = tmp / "home"
        tf_dir = tmp / "tf"
        bin_dir.mkdir()
        (home / ".oci-context").mkdir(parents=True)
        (home / ".oci").mkdir()
        (home / ".ssh").mkdir()
        tf_dir.mkdir()
        cache_dir = home / ".cache" / "bastion-session"
        cache_dir.mkdir(parents=True)

        resolved_oci_context = shutil_which_or_path(oci_context)
        resolved_bastion_session = shutil_which_or_path(bastion_session)
        if not resolved_oci_context or not resolved_bastion_session:
            print("OCI_CONTEXT_BIN and BASTION_SESSION_BIN must resolve to executables", file=sys.stderr)
            return 2
        copy_or_link(resolved_oci_context, bin_dir / "oci-context")
        copy_or_link(resolved_bastion_session, bin_dir / "bastion-session")

        write_exe(bin_dir / "oci", """#!/usr/bin/env python3
import json, sys
args=sys.argv[1:]
if args[:1] == ['--version']:
    print('3.99.0'); raise SystemExit(0)
out=[]; i=0
while i < len(args):
    if args[i] in ['--profile','--region','--auth','--config-file'] and i+1 < len(args):
        i += 2
    else:
        out.append(args[i]); i += 1
args=out
session={'id':'ocid1.bastionsession.oc1..fake','bastionId':'ocid1.bastion.oc1..b1','targetResourceId':'ocid1.instance.oc1..i1','targetResourceDetails':{'privateIpAddress':'10.42.1.217'},'lifecycleState':'ACTIVE','timeCreated':'2026-05-18T23:00:00Z','timeExpires':'2099-01-01T00:00:00Z'}
if args[:3] == ['iam','region-subscription','list']:
    print(json.dumps({'data':[{'is-home-region':True,'region-key':'IAD','region-name':'us-ashburn-1','status':'READY'}]})); raise SystemExit(0)
if args[:3] == ['bastion','session','list']:
    print('[]'); raise SystemExit(0)
if args[:3] == ['bastion','session','create-managed-ssh']:
    print(json.dumps(session)); raise SystemExit(0)
if args[:3] == ['bastion','session','get']:
    print(json.dumps(session)); raise SystemExit(0)
print('unexpected oci ' + ' '.join(sys.argv[1:]), file=sys.stderr); raise SystemExit(2)
""")

        write_exe(bin_dir / "ssh", """#!/bin/sh
if [ "$1" = "-G" ]; then
  echo "user opc"
  echo "hostname 10.42.1.217"
  echo "proxyjump DEFAULT-bastion"
  echo "identityfile ~/.ssh/fake.key"
  exit 0
fi
exit 2
""")

        (home / ".ssh" / "id_ed25519.pub").write_text("ssh-ed25519 AAA fake\n")
        (home / ".ssh" / "id_ed25519").write_text("private\n")
        (home / ".oci-context" / "config.yml").write_text(f"""options:
  oci_config_path: {home}/.oci/config
  socket_path: {home}/.oci-context/daemon.sock
contexts:
  - name: dev
    profile: DEFAULT
    auth_method: api_key
    tenancy_ocid: ocid1.tenancy.oc1..t
    compartment_ocid: ocid1.compartment.oc1..c
    region: us-ashburn-1
    user: ocid1.user.oc1..u
current_context: dev
""")
        (tf_dir / "outputs.json").write_text(json.dumps({
            "bastion_id": {"value": "ocid1.bastion.oc1..b1"},
            "instance_id": {"value": "ocid1.instance.oc1..i1"},
            "private_ip": {"value": "10.42.1.217"},
        }))
        (cache_dir / "current-bastion.json").write_text(json.dumps({
            "id": "ocid1.bastion.oc1..b1",
            "name": "b1",
            "region": "us-ashburn-1",
            "profile": "DEFAULT",
            "auth_method": "api_key",
            "context_name": "dev",
            "source": "test",
            "selected_at": "2026-05-19T00:00:00Z",
        }))

        env = os.environ.copy()
        env["PATH"] = str(bin_dir) + os.pathsep + env["PATH"]
        env["HOME"] = str(home)
        helper = helper_cmd(tmp)

        checks = [
            ("oci-bassh-track.schema.json", helper + ["track-from-terraform", "vmordws02", str(tf_dir)]),
            ("oci-bassh-track.schema.json", helper + ["track", "vmordws02", str(tf_dir)]),
            ("oci-bassh-track.schema.json", helper + ["track", "vmordws02", "--terraform-dir", str(tf_dir)]),
            ("oci-bassh-ensure.schema.json", helper + ["ensure-target", "vmordws02"]),
            ("oci-bassh-ensure.schema.json", helper + ["ensure", "vmordws02"]),
            ("oci-bassh-ssh.schema.json", helper + ["ssh", "--dry-run", "vmordws02"]),
            ("oci-bassh-doctor.schema.json", helper + ["doctor"]),
            ("oci-bassh-check.schema.json", helper + ["check"]),
            ("oci-bassh-inspect.schema.json", helper + ["inspect", "vmordws02"]),
            ("oci-bassh-repair.schema.json", helper + ["repair", "vmordws02"]),
            ("oci-bassh-repair.schema.json", helper + ["repair", "--ensure", "vmordws02"]),
            ("oci-bassh-paths.schema.json", helper + ["paths", "-o", "json"]),
            ("oci-bassh-upgrade.schema.json", helper + ["upgrade"]),
            ("oci-bassh-version.schema.json", helper + ["version", "-o", "json"]),
            ("oci-bassh-version.schema.json", helper + ["--version", "--json"]),
        ]
        for schema, cmd in checks:
            proc = run(cmd, env)
            if proc.returncode != 0:
                return proc.returncode
            data = json.loads(proc.stdout)
            validate(schema, data)
            if not data.get("ok"):
                print(proc.stdout)
                return 1
        print("real-binary e2e passed")
        return 0


def shutil_which_or_path(value):
    path = Path(value)
    if path.exists():
        return str(path)
    import shutil
    return shutil.which(value)


if __name__ == "__main__":
    raise SystemExit(main())
