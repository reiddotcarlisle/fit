# Project build & quality gates Makefile

BIN=bin
LIB=lib
PKG=./...
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-X 'main.buildVersion=$(VERSION)'

.PHONY: all build clean fit fit-hello libs lint tools fmt-check tidy-check vuln verify copy-libs test test-short test-race coverage coverage-html mod-verify generate-check release help

all: build

build: fit fit-hello copy-libs

fit:
	@mkdir -p $(BIN)
	go build -ldflags "$(LDFLAGS)" -o $(BIN)/fit ./cmd/fit

fit-hello:
	@mkdir -p $(BIN)
	- go build -ldflags "$(LDFLAGS)" -o $(BIN)/fit-hello ./cmd/fit-hello

copy-libs:
	@if ls $(LIB)/*.dll >/dev/null 2>&1; then cp $(LIB)/*.dll $(BIN)/; fi

fmt-check:
	@echo 'Checking formatting'
	@test -z "$$(gofmt -l . | grep -v '^vendor/')" || (echo "Run: gofmt -w (files need formatting)"; gofmt -l . | grep -v '^vendor/'; exit 1)

tidy-check:
	@echo 'Checking go.mod/go.sum tidy state'
	@cp go.mod go.mod.bak; cp go.sum go.sum.bak || true
	@go mod tidy
	@diff -q go.mod go.mod.bak >/dev/null || (echo "go.mod not tidy"; diff -u go.mod.bak go.mod; mv go.mod.bak go.mod; mv go.sum.bak go.sum; exit 1)
	@diff -q go.sum go.sum.bak >/dev/null || (echo "go.sum not tidy"; diff -u go.sum.bak go.sum; mv go.mod.bak go.mod || true; mv go.sum.bak go.sum; exit 1)
	@rm -f go.mod.bak go.sum.bak

lint: tools fmt-check tidy-check
	go vet $(PKG)
	@if command -v staticcheck >/dev/null 2>&1; then echo 'Running staticcheck'; staticcheck $(PKG); else echo 'staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)'; fi

vuln:
	@if command -v govulncheck >/dev/null 2>&1; then govulncheck $(PKG); else echo 'govulncheck not installed (go install golang.org/x/vuln/cmd/govulncheck@latest)'; fi

test:
	go test -count=1 $(PKG)

test-short:
	go test -short -count=1 $(PKG)

test-race:
	go test -race -count=1 $(PKG)

coverage:
	go test -cover -coverprofile=coverage.out $(PKG)
	@go tool cover -func=coverage.out | grep total

coverage-html: coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Open coverage.html in a browser."

mod-verify:
	go mod verify

generate-check:
	@echo 'Checking go generate reproducibility'
	@if grep -R "//go:generate" -n . >/dev/null 2>&1; then \
		git diff --quiet || (echo "Working tree dirty before generate"; exit 1); \
		go generate ./...; \
		git diff --quiet || (echo "Generated code not committed"; git --no-pager diff; exit 1); \
	else echo 'No go:generate directives'; fi

verify: lint test-short vuln mod-verify generate-check
	@echo 'All quality gates passed.'

tools:
	@echo 'Ensuring tool dependencies (tracked via tools.go)'
	@go list -deps ./... >/dev/null 2>&1 || true

release: verify
	@git diff --quiet || (echo "Uncommitted changes present"; exit 1)
	@echo "Ready to tag and release version $(VERSION)"

clean:
	rm -rf $(BIN) coverage.out coverage.html

help:
	@echo "Targets: build lint test test-race coverage vuln verify release clean"
