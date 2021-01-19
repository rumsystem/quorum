1. run the bootstrap node: ```./scripts/runbootstrap.sh```
2. run peer1: ```go run cmd/main.go -peername peer1 -listen /ip4/127.0.0.1/tcp/7001 -peer /ip4/127.0.0.1/tcp/10666/p2p/<Bootstrap node ID> -logtostderr=true```
3. run peer2: ```go run cmd/main.go -peername peer2 -listen /ip4/127.0.0.1/tcp/7002 -peer /ip4/127.0.0.1/tcp/10666/p2p/<Bootstrap node ID> -logtostderr=true```
