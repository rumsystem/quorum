PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = $(shell which protoc)
QUORUM_BIN_NAME=quorum
QUORUM_WASMLIB_NAME=lib.wasm
CLI_BIN_NAME=rumcli
GIT_COMMIT=$(shell git rev-list -1 HEAD)
LDFLAGS = -ldflags "-X main.GitCommit=${GIT_COMMIT}"
GOARCH = amd64

compile: chain.proto activity_stream.proto rumexchange.proto

chain.proto:
	protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/chain.proto
	mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/chain.pb.go internal/pkg/pb/chain.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' internal/pkg/pb/chain.pb.go

activity_stream.proto:
	protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/activity_stream.proto
	mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/activity_stream.pb.go internal/pkg/pb/activity_stream.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' internal/pkg/pb/activity_stream.pb.go

rumexchange.proto:
	protoc -I=internal/pkg/pb --go_out=internal/pkg/pb internal/pkg/pb/rumexchange.proto 
	mv internal/pkg/pb/github.com/rumsystem/quorum/internal/pkg/pb/rumexchange.pb.go internal/pkg/pb/rumexchange.pb.go
	sed -i 's/TimeStamp,omitempty/TimeStamp,omitempty,string/g' internal/pkg/pb/rumexchange.pb.go

linux:
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/linux_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go

freebsd:
	CGO_ENABLED=0 GO111MODULE=on GOOS=freebsd GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/freebsd_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go

darwin:
	CGO_ENABLED=0 GO111MODULE=on GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS}  -o dist/darwin_${GOARCH}/${QUORUM_BIN_NAME} cmd/main.go

windows:
	CGO_ENABLED=0 GO111MODULE=on GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o dist/windows_${GOARCH}/${QUORUM_BIN_NAME}.exe cmd/main.go

wasm:
	CGO_ENABLED=0 GO111MODULE=on GOOS=js GOARCH=wasm go build ${LDFLAGS} -o dist/js_wasm/${QUORUM_WASMLIB_NAME} cmd/wasm/lib.go

build: compile linux freebsd darwin windows wasm

buildall: linux freebsd darwin windows wasm

doc: 
	$(shell which swag) init -g ./cmd/main.go --parseDependency --parseInternal --parseDepth 2

test-main: 
	go test -timeout 99999s cmd/main_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20

test-main-rex: 
	go test -timeout 99999s cmd/main_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest true

test-api: 
	go test -v internal/pkg/api/*

test: test-api test-main test-main-rex

all: compile doc test buildall
