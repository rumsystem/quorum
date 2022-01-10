$env:RUM_KSPASSWD='password'


go run cmd/main.go -peername peer2 -listen /ip4/127.0.0.1/tcp/7003/ws -apilisten :8003 -peer /ip4/127.0.0.1/tcp/10666/ws/p2p/16Uiu2HAmPfKKtxsvVuhfhjq8SVGWpGQQTusB6zhK6Nc78u4bwFgx -configdir config/peer2 -datadir data/peer2 -keystoredir keystore/peer2 -debug true
