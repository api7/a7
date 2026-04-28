BINARY  := a7
MODULE  := github.com/api7/a7
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: build test test-verbose lint fmt vet check install clean docker-up docker-down validate-skills test-skills test-e2e test-e2e-full

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/a7

test:
	go test ./... -count=1

test-verbose:
	go test ./... -count=1 -v

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w .

vet:
	go vet ./...

check: fmt vet lint test validate-skills test-skills

validate-skills:
	./scripts/validate-skills.sh

test-skills:
	go test ./test/e2e/skills -tags=e2e -count=1

install: build
	cp bin/$(BINARY) $(GOPATH)/bin/$(BINARY)

clean:
	rm -rf bin/

docker-up:
	docker compose -f test/e2e/docker-compose.yml up -d --wait --wait-timeout 120

docker-down:
	docker compose -f test/e2e/docker-compose.yml down -v

# test-e2e and test-e2e-full intentionally share the same Ginkgo entrypoint.
# The default CI stays narrower via runtime guards such as requireGatewayURL,
# requireHTTPBin, and A7_E2E_ENABLE_GATEWAY_GROUP_CRUD.
test-e2e:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --procs=1 --tags=e2e --timeout=45m ./test/e2e/...

test-e2e-full:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --procs=1 --tags=e2e --timeout=45m ./test/e2e/...
