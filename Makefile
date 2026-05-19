.PHONY: build test e2e

build:
	go build -o oci-hop ./cmd/oci-hop

test:
	go test ./...

e2e: test
	go test -tags=e2e ./...
