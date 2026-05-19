.PHONY: build test e2e

build:
	go build ./cmd/oci-bassh

test:
	go test ./...

e2e: test
	go test -tags=e2e ./...
