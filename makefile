PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = $(shell which protoc)
QUORUM_BIN_NAME=quorum
QUORUM_WASMLIB_NAME=lib.wasm
CLI_BIN_NAME=rumcli
GIT_COMMIT=$(shell git rev-list -1 HEAD)
LDFLAGS = -ldflags "-s -w -X main.GitCommit=${GIT_COMMIT}"

export GOARCH = amd64
export CGO_ENABLED = 0
export GO111MODULE = on

define build-quorum
go build ${LDFLAGS} -o dist/${GOOS}_${GOARCH}/${QUORUM_BIN_NAME} main.go
endef

define build-wasm
go build ${LDFLAGS} -o dist/${GOOS}_${GOARCH}/${QUORUM_WASMLIB_NAME} cmd/wasm/lib.go
endef

define build-cli
go build ${LDFLAGS} -o dist/${GOOS}_${GOARCH}/${CLI_BIN_NAME} cmd/cli/main.go
endef

linux: export GOOS = linux
linux:
	$(build-quorum)

freebsd: export GOOS = freebsd
freebsd:
	$(build-quorum)

darwin: export GOOS = darwin
darwin:
	$(build-quorum)

windows: export GOOS = windows
windows: QUORUM_BIN_NAME = quorum.exe
windows:
	$(build-quorum)

wasm: export GOOS = js
wasm: export GOARCH = wasm
wasm:
	$(build-wasm)

cli_linux: export GOOS = linux
cli_linux:
	$(build-cli)

cli_freebsd: export GOOS = freebsd
cli_freebsd:
	$(build-cli)

cli_darwin: export GOOS = darwin
cli_darwin:
	$(build-cli)

cli_windows: export GOOS = windows
cli_windows: CLI_BIN_NAME = rumcli.exe
cli_windows:
	$(build-cli)

buildcli: cli_linux cli_freebsd cli_darwin cli_windows

build: linux freebsd darwin windows wasm

buildall: build buildcli

install-swag:
	go install github.com/swaggo/swag/cmd/swag@latest

gen-doc: install-swag
	$(shell which swag) init -g main.go --parseDependency --parseInternal --parseDepth 3 --parseGoList=false

serve-doc: gen-doc
	go run ./cmd/docs.go

test-main:
	go test -timeout 99999s main_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20

test-main-rex:
	go test -timeout 99999s main_rex_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true

test-api:
	go test -v pkg/chainapi/api/*

test: test-api test-main test-main-rex

all: doc test buildall
