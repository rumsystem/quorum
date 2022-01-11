#!/bin/bash

GIT_COMMIT=$(git rev-list -1 HEAD)
#darwin windows freebsd linux
for GOOS in linux ; do
    for GOARCH in amd64; do
        if [[ "$GOOS" == "windows" ]]; then
            bin="quorum.exe"
        else
            bin="quorum"
        fi
        env CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
            go build -ldflags "-X main.GitCommit=$GIT_COMMIT" \
            -o dist/${GOOS}_${GOARCH}/$bin cmd/main.go
    done
done
