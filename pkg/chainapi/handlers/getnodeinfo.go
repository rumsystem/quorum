package handlers

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/rumsystem/quorum/internal/pkg/conn/p2p"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/internal/pkg/utils"
)

type NodeInfo struct {
	NodeID        string              `json:"node_id" validate:"required" example:"16Uiu2HAkytdk8dhP8Z1JWvsM7qYPSLpHxLCfEWkSomqn7Tj6iC2d"`
	NodePublickey string              `json:"node_publickey" validate:"required" example:"CAISIQJCVubdxsT/FKvnBT9r68W4Nmh0/2it7KY+dA7x25NtYg=="`
	NodeStatus    string              `json:"node_status" validate:"required" example:"NODE_ONLINE"`
	NodeType      string              `json:"node_type" validate:"required" example:"peer"`
	NodeVersion   string              `json:"node_version" validate:"required" example:"1.0.0 - 99bbd8e65105c72b5ca57e94ae5be117eaf05f0d"`
	Peers         map[string][]string `json:"peers" validate:"required"` // Example: {"/quorum/nevis/meshsub/1.1.0": ["16Uiu2HAmM4jFjs5EjakvGgJkHS6Lg9jS6miNYPgJ3pMUvXGWXeTc"]}
	Mem           NodeInfoMem         `json:"mem"`
}

type ByteSize uint64
type NodeInfoMem struct {
	Sys        ByteSize `json:"sys"` // OS memory being used
	HeapSys    ByteSize `json:"heap_sys"`
	HeapAlloc  ByteSize `json:"heap_alloc"`
	HeapInuse  ByteSize `json:"heap_inuse"`
	StackSys   ByteSize `json:"stack_sys"`
	StackInuse ByteSize `json:"stack_inuse"`
	NumGC      uint32   `json:"num_gc"`
}

func (v ByteSize) MarshalJSON() ([]byte, error) {
	i := int64(v)
	s := b2m(i)
	return json.Marshal(s)
}

func b2m(i int64) string {
	const unit = 1024

	if i < unit {
		return fmt.Sprintf("%d B", i)
	}

	div, exp := int64(unit), 0
	for n := i / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(i)/float64(div), "KMGTPE"[exp])
}

func updateNodeStatus(nodenetworkname string) {
	peersprotocol := nodectx.GetNodeCtx().PeersProtocol()
	networknamewithprefix := fmt.Sprintf("%s/%s", p2p.ProtocolPrefix, nodenetworkname)
	for protocol, peerlist := range *peersprotocol {
		if strings.HasPrefix(protocol, networknamewithprefix) {
			if len(peerlist) > 0 {
				nodectx.GetNodeCtx().UpdateOnlineStatus(nodectx.NODE_ONLINE)
				return
			}
		}
	}
	if nodectx.GetNodeCtx().Status != nodectx.NODE_OFFLINE {
		nodectx.GetNodeCtx().UpdateOnlineStatus(nodectx.NODE_OFFLINE)
	}
}

func GetNodeInfo(networkName string) (*NodeInfo, error) {
	var info NodeInfo

	info.NodeVersion = nodectx.GetNodeCtx().Version + " - " + utils.GitCommit
	info.NodeType = "peer"
	updateNodeStatus(networkName)

	if nodectx.GetNodeCtx().Status == nodectx.NODE_ONLINE {
		info.NodeStatus = "NODE_ONLINE"
	} else {
		info.NodeStatus = "NODE_OFFLINE"
	}

	pubkeybytes, err := p2pcrypto.MarshalPublicKey(nodectx.GetNodeCtx().PublicKey)
	if err != nil {
		return nil, err
	}

	info.NodePublickey = p2pcrypto.ConfigEncodeKey(pubkeybytes)
	info.NodeID = nodectx.GetNodeCtx().PeerId.Pretty()

	peers := nodectx.GetNodeCtx().PeersProtocol()
	info.Peers = *peers

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	info.Mem = NodeInfoMem{
		Sys:        ByteSize(m.Sys),
		HeapSys:    ByteSize(m.HeapSys),
		HeapAlloc:  ByteSize(m.HeapAlloc),
		HeapInuse:  ByteSize(m.HeapInuse),
		StackSys:   ByteSize(m.StackSys),
		StackInuse: ByteSize(m.StackInuse),
		NumGC:      m.NumGC,
	}

	return &info, nil
}
