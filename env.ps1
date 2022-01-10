$env:GOOS="js"
$env:GOARCH="wasm"
$env:GO111MODULE="on"
$env:GIT_COMMIT=$(git rev-parse --short HEAD)