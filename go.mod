module github.com/rumsystem/quorum

go 1.15

replace github.com/libp2p/go-libp2p-autonat => github.com/huo-ju/go-libp2p-autonat v0.4.3

replace github.com/dgraph-io/badger/v3 => github.com/chux0519/badger/v3 v3.2103.3

replace github.com/dgraph-io/ristretto => github.com/chux0519/ristretto v0.1.1

require (
	code.rocketnine.space/tslocum/cbind v0.1.5
	code.rocketnine.space/tslocum/cview v1.5.6
	filippo.io/age v1.0.0-rc.3
	github.com/BurntSushi/toml v0.3.1
	github.com/Press-One/go-update v1.0.0
	github.com/adrg/xdg v0.3.3
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/dgraph-io/badger/v3 v3.2011.1
	github.com/ethereum/go-ethereum v1.10.8
	github.com/gdamore/tcell/v2 v2.4.0
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/validator/v10 v10.5.0
	github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/golang/protobuf v1.5.2
	github.com/google/orderedcode v0.0.1
	github.com/google/uuid v1.2.0
	github.com/gopherjs/gopherjs v0.0.0-20190812055157-5d271430af9f // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/huo-ju/quercus v0.0.0-20210909192534-3740345b9ab8
	github.com/ipfs/go-ds-badger2 v0.1.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/labstack/echo/v4 v4.3.0
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libp2p/go-libp2p v0.14.2
	github.com/libp2p/go-libp2p-connmgr v0.2.4
	github.com/libp2p/go-libp2p-core v0.8.6
	github.com/libp2p/go-libp2p-discovery v0.5.1
	github.com/libp2p/go-libp2p-kad-dht v0.11.1
	github.com/libp2p/go-libp2p-peerstore v0.2.8
	github.com/libp2p/go-libp2p-pubsub v0.5.4
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spf13/viper v1.7.1
	github.com/swaggo/echo-swagger v1.1.0
	github.com/swaggo/swag v1.7.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.18.1 // indirect
	golang.org/x/crypto v0.0.0-20210616213533-5ff15b29337e // indirect
	golang.org/x/lint v0.0.0-20201208152925-83fdc39ff7b5 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b
	golang.org/x/tools v0.1.4 // indirect
	google.golang.org/grpc v1.35.0 // indirect
	google.golang.org/protobuf v1.26.0
)
