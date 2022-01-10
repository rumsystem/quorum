$env:RUM_KSPASSWD='password'


go run cmd/main.go -peername peer1 -listen /ip4/127.0.0.1/tcp/7002/ws -apilisten :8002 -peer /ip4/127.0.0.1/tcp/10666/ws/p2p/16Uiu2HAmPfKKtxsvVuhfhjq8SVGWpGQQTusB6zhK6Nc78u4bwFgx -configdir config/peer1 -datadir data/peer1 -keystoredir keystore/peer1 -debug true
