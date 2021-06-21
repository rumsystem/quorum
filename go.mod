module github.com/huo-ju/quorum

go 1.15

//replace github.com/ipfs/go-ipfs-blockstore => github.com/huo-ju/go-ipfs-blockstore v1.0.5

require (
	github.com/RichardKnop/machinery v1.10.5
	github.com/dgraph-io/badger v1.6.1
	github.com/dgraph-io/badger/v3 v3.2011.1
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator v9.31.0+incompatible
	github.com/go-playground/validator/v10 v10.5.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.3
	github.com/google/uuid v1.2.0
	github.com/huo-ju/go-ipfs-blockstore v1.0.5
	github.com/ipfs/go-bitswap v0.3.3
	github.com/ipfs/go-block-format v0.0.3
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-badger v0.2.3
	github.com/ipfs/go-graphsync v0.6.0
	github.com/ipfs/go-log v1.0.4
	github.com/ipfs/go-log/v2 v2.1.1
	github.com/labstack/echo/v4 v4.1.17
	github.com/labstack/gommon v0.3.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libp2p/go-libp2p v0.13.0
	github.com/libp2p/go-libp2p-autonat v0.4.0
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.0
	github.com/libp2p/go-libp2p-discovery v0.5.0
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/libp2p/go-libp2p-kbucket v0.4.7
	github.com/libp2p/go-libp2p-pubsub v0.4.1
	github.com/libp2p/go-msgio v0.0.6
	github.com/mr-tron/base58 v1.2.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/oklog/ulid v1.3.1
	github.com/spf13/viper v1.7.1
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.1.0 // indirect
	google.golang.org/protobuf v1.25.0
)
