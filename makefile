PROTOC_GEN_GO = $(GOPATH)/bin/protoc-gen-go
PROTOC = $(shell which protoc)
APP_NAME ?= quorum
DIST_DIR ?= dist
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo devel)
LDFLAGS ?= -s -w -X main.GitCommit=$(GIT_COMMIT)
CGO_ENABLED ?= 0
GORELEASER_VERSION ?= v2.17.0-d9421c5f-nightly
SWAG_VERSION ?= v1.16.6

.PHONY: build linux windows freebsd darwin buildall install-goreleaser goreleaser-build goreleaser-build-all release install-swag gen-doc serve-doc test-main test-main-rex test-api test all

install-goreleaser:
	go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

build:
	GOOS= GOARCH= CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME) .

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)_linux_amd64/$(APP_NAME) .

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)_windows_amd64/$(APP_NAME).exe .

freebsd:
	GOOS=freebsd GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)_freebsd_amd64/$(APP_NAME) .

darwin:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(APP_NAME)_darwin_amd64/$(APP_NAME) .

buildall: linux windows freebsd darwin

goreleaser-build: install-goreleaser
	GOOS=linux GOARCH=amd64 goreleaser build --snapshot --clean --single-target

goreleaser-build-all: install-goreleaser
	goreleaser build --snapshot --clean

release: install-goreleaser
	goreleaser release --clean

install-swag:
	go install github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)

gen-doc: install-swag
	$(shell which swag) init -g main.go --parseDependency --parseInternal --parseDepth 3 --parseGoList=true

serve-doc: gen-doc
	go run -tags docs ./docs.go

test-main:
	go test -timeout 99999s main_test.go -v

test-main-rex:
	#go test -timeout 99999s main_rex_test.go -v -nodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true
	go test -timeout 99999s main_test.go -v -fullnodes=3 -posts=2 -timerange=5 -groups=3 -synctime=20 -rextest=true

test-api:
	go test -v pkg/chainapi/api/*

test: test-api test-main test-main-rex

all: doc test buildall release
