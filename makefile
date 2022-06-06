PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = $(shell which protoc)
QUORUM_BIN_NAME=quorum
LIGHT_QUORUM_BIN_NAME=lightquorum
QUORUM_WASMLIB_NAME=lib.wasm
CLI_BIN_NAME=rumcli
GIT_COMMIT=$(shell git rev-list -1 HEAD)
LDFLAGS = -ldflags "-X main.GitCommit=${GIT_COMMIT}"
GOARCH = amd64

linux:
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/linux_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/linux_${GOARCH}/${LIGHT_QUORUM_BIN_NAME} pkg/nodesdk/cmd/main.go

freebsd:
	CGO_ENABLED=0 GO111MODULE=on GOOS=freebsd GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/freebsd_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go
	CGO_ENABLED=0 GO111MODULE=on GOOS=freebsd GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/freebsd_${GOARCH}/${LIGHT_QUORUM_BIN_NAME} pkg/nodesdk/cmd/main.go

darwin:
	CGO_ENABLED=0 GO111MODULE=on GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS}  -o dist/darwin_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go
	CGO_ENABLED=0 GO111MODULE=on GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/darwin_${GOARCH}/${LIGHT_QUORUM_BIN_NAME} pkg/nodesdk/cmd/main.go

windows:
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/windows_${GOARCH}/${QUORUM_BIN_NAME}.exe cmd/main.go
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/windows_${GOARCH}/${LIGHT_QUORUM_BIN_NAME} pkg/nodesdk/cmd/main.go

wasm:
	CGO_ENABLED=0 GO111MODULE=on GOOS=js GOARCH=wasm go build ${LDFLAGS} -o dist/js_wasm/${QUORUM_WASMLIB_NAME} cmd/wasm/lib.go


cli_linux:
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/linux_${GOARCH}/${CLI_BIN_NAME} cmd/cli/main.go

cli_freebsd:
	CGO_ENABLED=0 GO111MODULE=on GOOS=freebsd GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/freebsd_${GOARCH}/${CLI_BIN_NAME} cmd/cli/main.go

cli_darwin:
	CGO_ENABLED=0 GO111MODULE=on GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS}  -o dist/darwin_${GOARCH}/${CLI_BIN_NAME} cmd/cli/main.go

cli_windows:
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/windows_${GOARCH}/${CLI_BIN_NAME} cmd/cli/main.go

buildcli: cli_linux cli_freebsd cli_darwin cli_windows

build: linux freebsd darwin windows wasm

buildall: build buildcli

doc: 
	$(shell which swag) init -g ./cmd/main.go --parseDependency --parseInternal --parseDepth 2

test-main: 
	go test -timeout 99999s cmd/main_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20

test-main-rex: 
	go test -timeout 99999s cmd/main_rex_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true

test-api: 
	go test -v internal/pkg/api/*

test: test-api test-main test-main-rex

all: doc test buildall
