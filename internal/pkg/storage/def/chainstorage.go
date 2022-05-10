package def

import (
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

type ChainStorageIface interface {
	DeleteRelay(relayid string) (bool, *quorumpb.GroupRelayItem, error)
	AddRelayActivity(groupRelayItem *quorumpb.GroupRelayItem) (string, error)
	AddRelayReq(groupRelayItem *quorumpb.GroupRelayItem) (string, error)
}
