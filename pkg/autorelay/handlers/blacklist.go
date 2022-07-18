package handlers

import (
	"strconv"

	"github.com/rumsystem/quorum/internal/pkg/storage"
)

type GetBlacklistResult struct {
	Peers []string `json:"peers"`
}

type AddBlacklistParam struct {
	FromPeer string `json:"from_peer"`
	ToPeer   string `json:"to_peer"`
}

type DelBlacklistParam AddBlacklistParam

type AddBlacklistResult struct {
	Ok bool `json:"ok"`
}

type DelBlacklistResult AddBlacklistResult

/* GetBlacklist to get blacklist for a server peer */
func GetBlacklist(db storage.QuorumStorage, serverPeer string) (*GetBlacklistResult, error) {
	res := &GetBlacklistResult{[]string{}}

	prefix := []byte(GetBlackListPrefixKey(serverPeer))
	if _, err := db.PrefixForeachKey(prefix, prefix, false, func(k []byte, err error) error {
		if err != nil {
			return err
		}
		key := string(k)
		banPeer := GetBlacklistPeerFromKeyByPrefix(key, string(prefix))
		res.Peers = append(res.Peers, banPeer)
		return nil
	}); err != nil {
		return nil, err
	}

	return res, nil
}

/* AddBlacklist to put a peer in blacklist */
func AddBlacklist(db storage.QuorumStorage, param AddBlacklistParam) (*AddBlacklistResult, error) {
	res := &AddBlacklistResult{true}

	k := []byte(GetBlackListKey(param.FromPeer, param.ToPeer))
	if err := db.Set(k, []byte(strconv.FormatBool(true))); err != nil {
		return nil, err
	}

	return res, nil
}

/* DeleteBlacklist to remove a peer from blacklist */
func DeleteBlacklist(db storage.QuorumStorage, param DelBlacklistParam) (*DelBlacklistResult, error) {
	res := &DelBlacklistResult{true}

	k := []byte(GetBlackListKey(param.FromPeer, param.ToPeer))
	if err := db.Delete(k); err != nil {
		return nil, err
	}

	return res, nil
}

func CheckBlacklist(db storage.QuorumStorage, from string, to string) (bool, error) {
	k := []byte(GetBlackListKey(from, to))

	return db.IsExist(k)
}
