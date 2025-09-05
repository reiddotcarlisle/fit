# Simple cross-platform convenience targets

BIN=bin
LIB=lib

.PHONY: all build clean fit fit-hello libs

all: build

build: fit fit-hello copy-libs

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
