.PHONY: build test e2e

build:
	go build ./cmd/oci-bassh

test:
	go test ./...
	python3 -m py_compile scripts/*.py

e2e: test
	python3 scripts/e2e_fake_cli.py
	python3 scripts/e2e_real_binaries.py
