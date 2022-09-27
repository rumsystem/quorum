module github.com/rumsystem/quorum

go 1.18

//replace github.com/libp2p/go-libp2p-autonat => github.com/huo-ju/go-libp2p-autonat v0.4.3

replace github.com/dgraph-io/badger/v3 => github.com/chux0519/badger/v3 v3.2103.3

replace github.com/dgraph-io/ristretto => github.com/chux0519/ristretto v0.1.1

//replace github.com/libp2p/go-libp2p-noise => github.com/libp2p/go-libp2p-noise v0.3.0

replace github.com/ipfs/go-ds-badger2 => github.com/ipfs/go-ds-badger2 v0.1.2

// replace github.com/libp2p/go-libp2p => github.com/libp2p/go-libp2p v0.19.0

//replace github.com/libp2p/go-libp2p-autonat => ../go-libp2p-autonat
//replace github.com/ipfs/go-ds-badger2 => ../go-ds-badger2
//replace github.com/libp2p/go-libp2p-pubsub => ../go-libp2p-pubsub
//replace github.com/libp2p/go-libp2p => github.com/chux0519/go-libp2p v0.19.1-0.20220814133711-afcc91c7c730
// replace github.com/libp2p/go-libp2p => ../go-libp2p

require (
	code.rocketnine.space/tslocum/cbind v0.1.5
	code.rocketnine.space/tslocum/cview v1.5.6
	filippo.io/age v1.0.0
	github.com/BurntSushi/toml v1.1.0
	github.com/Press-One/go-update v1.0.0
	github.com/adrg/xdg v0.3.3
	github.com/atotto/clipboard v0.1.4
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/ethereum/go-ethereum v1.10.17
	github.com/gdamore/tcell/v2 v2.4.0
	github.com/go-playground/locales v0.13.0
	github.com/go-playground/universal-translator v0.17.0
	github.com/go-playground/validator/v10 v10.5.0
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/protobuf v1.5.2
	github.com/google/go-cmp v0.5.8
	github.com/google/orderedcode v0.0.1
	github.com/google/uuid v1.3.0
	github.com/hack-pad/go-indexeddb v0.2.0
	github.com/huo-ju/quercus v0.0.0-20210909192534-3740345b9ab8
	github.com/ipfs/go-ds-badger2 v0.1.0
	github.com/ipfs/go-ipfs-util v0.0.2
	github.com/ipfs/go-log/v2 v2.5.1
	github.com/labstack/echo/v4 v4.7.2
	github.com/labstack/gommon v0.3.1
	github.com/libp2p/go-libp2p v0.23.2
	github.com/libp2p/go-libp2p-kad-dht v0.18.0
	github.com/libp2p/go-libp2p-pubsub v0.8.1
	github.com/libp2p/go-msgio v0.2.0
	github.com/multiformats/go-multiaddr v0.7.0
	github.com/open-policy-agent/opa v0.39.0
	github.com/otiai10/copy v1.7.0
	github.com/phayes/freeport v0.0.0-20220201140144-74d24b5ae9f5
	github.com/rumsystem/ip-cert v0.0.0-20220802012323-cebacb66b383
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.12.0
	github.com/swaggo/echo-swagger v1.1.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	google.golang.org/protobuf v1.28.1
)

require (
	github.com/NebulousLabs/errors v0.0.0-20181203160057-9f787ce8f69e // indirect
	github.com/NebulousLabs/fastrand v0.0.0-20181203155948-6fb6489aac4e // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/ipfs/go-ipns v0.2.0 // indirect
	github.com/libp2p/go-libp2p-asn-util v0.2.0 // indirect
	github.com/libp2p/go-libp2p-core v0.20.0 // indirect
	github.com/libp2p/go-libp2p-kbucket v0.4.7 // indirect
	github.com/libp2p/go-libp2p-record v0.2.0 // indirect
	github.com/libp2p/go-libp2p-routing-helpers v0.2.3 // indirect
	github.com/libp2p/go-yamux/v4 v4.0.0 // indirect
	github.com/marten-seemann/qtls-go1-19 v0.1.0 // indirect
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/swaggo/swag v1.8.1 // indirect
	golang.org/x/exp v0.0.0-20220916125017-b168a2c6b86b // indirect
)

require (
	github.com/DataDog/zstd v1.4.1 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/NebulousLabs/merkletree v0.0.0-20181203152040-08d5d54b07f5
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.2.0 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/containerd/cgroups v1.0.4 // indirect
	github.com/coreos/go-systemd/v22 v22.4.0 // indirect
	github.com/davidlazar/go-crypto v0.0.0-20200604182044-b73af7476f6c // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0
	github.com/dgraph-io/badger/v2 v2.2007.3 // indirect
	github.com/dgraph-io/ristretto v0.1.0 // indirect
	github.com/dgryski/go-farm v0.0.0-20200201041132-a6ae2369ad13 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/elastic/gosigar v0.14.2 // indirect
	github.com/flynn/noise v1.0.0 // indirect
	github.com/francoispqt/gojay v1.2.13 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.6 // indirect
	github.com/go-openapi/spec v0.20.4 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/go-playground/form/v4 v4.2.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt v3.2.2+incompatible // indirect
	github.com/golang/glog v1.0.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/flatbuffers v1.12.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gopacket v1.1.19 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20190812055157-5d271430af9f // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru v0.5.5-0.20210104140557-80c98217689d // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huin/goupnp v1.0.3 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/ipfs/go-cid v0.3.2 // indirect
	github.com/ipfs/go-datastore v0.6.0 // indirect
	github.com/ipfs/go-log v1.0.5 // indirect
	github.com/ipld/go-ipld-prime v0.9.0 // indirect
	github.com/jackpal/go-nat-pmp v1.0.2 // indirect
	github.com/jbenet/go-temp-err-catcher v0.1.0 // indirect
	github.com/jbenet/goprocess v0.1.4 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.15.10 // indirect
	github.com/klauspost/cpuid/v2 v2.1.1 // indirect
	github.com/klauspost/reedsolomon v1.10.0
	github.com/koron/go-ssdp v0.0.3 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
	github.com/libp2p/go-cidranger v1.1.0 // indirect
	github.com/libp2p/go-flow-metrics v0.1.0 // indirect
	github.com/libp2p/go-nat v0.1.0 // indirect
	github.com/libp2p/go-netroute v0.2.0 // indirect
	github.com/libp2p/go-openssl v0.1.0 // indirect
	github.com/libp2p/go-reuseport v0.2.0 // indirect
	github.com/lucas-clemente/quic-go v0.29.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.6 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/marten-seemann/qtls-go1-18 v0.1.2 // indirect
	github.com/marten-seemann/tcp v0.0.0-20210406111302-dfbc87cc63fd // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/miekg/dns v1.1.50 // indirect
	github.com/mikioh/tcpinfo v0.0.0-20190314235526-30a79bb1804b // indirect
	github.com/mikioh/tcpopt v0.0.0-20190314235656-172688c1accc // indirect
	github.com/minio/sha256-simd v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-base32 v0.1.0 // indirect
	github.com/multiformats/go-base36 v0.1.0 // indirect
	github.com/multiformats/go-multiaddr-dns v0.3.1 // indirect
	github.com/multiformats/go-multiaddr-fmt v0.1.0 // indirect
	github.com/multiformats/go-multibase v0.1.1 // indirect
	github.com/multiformats/go-multicodec v0.6.0 // indirect
	github.com/multiformats/go-multihash v0.2.1 // indirect
	github.com/multiformats/go-multistream v0.3.3 // indirect
	github.com/multiformats/go-varint v0.0.6 // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/opencontainers/runtime-spec v1.0.2 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.0.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/polydawn/refmt v0.0.0-20190807091052-3d65705ee9f1 // indirect
	github.com/prometheus/client_golang v1.13.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.37.0 // indirect
	github.com/prometheus/procfs v0.8.0 // indirect
	github.com/raulk/go-watchdog v1.3.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0 // indirect
	github.com/rivo/uniseg v0.3.4 // indirect
	github.com/rjeczalik/notify v0.9.2 // indirect
	github.com/smartystreets/assertions v1.0.1 // indirect
	github.com/spacemonkeygo/spacelog v0.0.0-20180420211403-2296661a0572 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.9.0 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.4.0 // indirect
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.1 // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/yashtewari/glob-intersection v0.1.0 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0 // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/net v0.0.0-20220920183852-bf014ff85ad5 // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4 // indirect
	golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab // indirect
	golang.org/x/term v0.0.0-20220411215600-e5f449aeb171
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210220033141-f8bda1e9f3ba // indirect
	golang.org/x/tools v0.1.12 // indirect
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	lukechampine.com/blake3 v1.1.7 // indirect
)
