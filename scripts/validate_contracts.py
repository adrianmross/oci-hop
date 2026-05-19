#!/usr/bin/env python3
import json
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCHEMAS = ROOT / "schemas"


REQUIRED = {
    "oci-bassh-doctor.schema.json": ["ok", "tools", "oci_context", "bastion_doctor", "targets"],
    "oci-bassh-ensure.schema.json": ["ok", "host", "auth", "ensure", "ssh_config", "connect_command"],
    "oci-bassh-track.schema.json": ["ok", "host", "track", "target"],
    "oci-bassh-ssh.schema.json": ["ok", "host", "auth", "ensure", "ssh_command"],
    "oci-bassh-contract-check.schema.json": ["ok", "checks"],
}


def load(path):
    with open(path) as f:
        return json.load(f)


def main():
    for schema in SCHEMAS.glob("*.schema.json"):
        load(schema)
    if len(sys.argv) < 3 or len(sys.argv[1:]) % 2:
        print("usage: validate_contracts.py <schema-name> <json-file> ...", file=sys.stderr)
        return 2
    for schema_name, json_file in zip(sys.argv[1::2], sys.argv[2::2]):
        payload = load(json_file)
        for key in REQUIRED.get(schema_name, []):
            if key not in payload:
                print(f"{json_file}: missing required key {key}", file=sys.stderr)
                return 1
        if "ok" in payload and not isinstance(payload["ok"], bool):
            print(f"{json_file}: ok must be boolean", file=sys.stderr)
            return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
