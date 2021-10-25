wasm demo

## env

```
export GOOS=js
export GOARCH=wasm
export GO111MODULE=on
cp $(go env GOROOT)/misc/wasm/wasm_exec.js .
```

## buid

> go build -o lib.wasm lib.go

## run

```
go run bootstrap.go

# /ip4/127.0.0.1/tcp/4000/ws/QmYSMod2mNuzueuHwuCV7Urj8JJqzYaMp4vB7jjDcWdtmG
```

in browser console:

```
StartQuorum("/ip4/127.0.0.1/tcp/10666/ws/p2p/16Uiu2HAkxcVyepsYE2YNTTr89ghBW6n6qpEe6tRrZGBJckV3oCZ3")
```
