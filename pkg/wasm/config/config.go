//go:build js && wasm
// +build js,wasm

package config

import ma "github.com/multiformats/go-multiaddr"

var DefaultRendezvousString = "e6629921-b5cd-4855-9fcd-08bcc39caef7"
var DefaultRoutingProtoPrefix = "/quorum/nevis"
var DefaultNetworkName = "nevis"
var DefaultPubSubProtocol = "/quorum/nevis/meshsub/1.1.0"

type BrowserConfig struct {
	BootstrapPeers     []ma.Multiaddr
	RendezvousString   string
	RoutingProtoPrefix string
	NetworkName        string
	UseTestNet         bool
	PubSubProtocol     string
}

func NewBrowserConfig(bootstrapStrs []string) *BrowserConfig {
	ret := BrowserConfig{}
	bootAddrs, _ := StringsToAddrs(bootstrapStrs)
	ret.BootstrapPeers = bootAddrs
	ret.RendezvousString = DefaultRendezvousString
	ret.RoutingProtoPrefix = DefaultRoutingProtoPrefix
	ret.NetworkName = DefaultNetworkName
	ret.UseTestNet = false
	ret.PubSubProtocol = DefaultPubSubProtocol
	return &ret
}

func StringsToAddrs(addrStrings []string) (maddrs []ma.Multiaddr, err error) {
	for _, addrString := range addrStrings {
		addr, err := ma.NewMultiaddr(addrString)
		if err != nil {
			println(err.Error())
			return maddrs, err
		}
		maddrs = append(maddrs, addr)
	}
	return
}
