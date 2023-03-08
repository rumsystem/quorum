PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = $(shell which protoc)

build:
	goreleaser build --snapshot --clean

buildall: build

install-goreleaser:
	go install github.com/goreleaser/goreleaser@latest

release: install-goreleaser
	goreleaser release --clean

install-swag:
	go install github.com/swaggo/swag/cmd/swag@v1.8.4

gen-doc: install-swag
	$(shell which swag) init -g main.go --parseDependency --parseInternal --parseDepth 3 --parseGoList=true

serve-doc: gen-doc
	go run ./docs.go

test-main:
	go test -timeout 99999s main_test.go -v

test-main-rex:
	#go test -timeout 99999s main_rex_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true
	go test -timeout 99999s main_test.go -v -fullnodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true

test-api:
	go test -v pkg/chainapi/api/*

test: test-api test-main test-main-rex

all: doc test buildall release
