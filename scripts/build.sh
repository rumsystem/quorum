export GIT_COMMIT=$(git rev-list -1 HEAD) && env GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -ldflags "-X main.GitCommit=$GIT_COMMIT" -o dist/linux_amd64/quorum cmd/main.go
export GIT_COMMIT=$(git rev-list -1 HEAD) && env GOOS=windows CGO_ENABLED=0 GOARCH=amd64 go build -ldflags "-X main.GitCommit=$GIT_COMMIT" -o dist/windows_amd64/quorum cmd/main.go
