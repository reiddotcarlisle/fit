# Simple cross-platform convenience targets

BIN=bin
LIB=lib

.PHONY: all build clean fit fit-hello libs lint tools

all: build

build: fit fit-hello copy-libs

lint: tools
	go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then echo 'Running staticcheck'; staticcheck ./...; else echo 'staticcheck not installed (go install honnef.co/go/tools/cmd/staticcheck@latest)'; fi

tools:
	@echo 'Ensuring tool dependencies (go mod tidy will keep them)'
	@go list -deps ./... >/dev/null

fit:
	@mkdir -p $(BIN)
	go build -o $(BIN)/fit ./cmd/fit

fit-hello:
	@mkdir -p $(BIN)
	- go build -o $(BIN)/fit-hello ./cmd/fit-hello

copy-libs:
	@if ls $(LIB)/*.dll >/dev/null 2>&1; then cp $(LIB)/*.dll $(BIN)/; fi

clean:
	rm -rf $(BIN)
