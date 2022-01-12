$env:RUM_KSPASSWD='password'
$env:CGO_ENABLED=0
go run cmd/main.go -bootstrap -listen /ip4/0.0.0.0/tcp/10666/ws -logtostderr=true