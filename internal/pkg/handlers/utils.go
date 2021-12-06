package handlers

import (
	"errors"
	"fmt"
	"strings"

	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
)

func initSignKey(groupId string, ks localcrypto.Keystore, nodeoptions *options.NodeOptions) (string, error) {
	hexkey, err := ks.GetEncodedPubkey(groupId, localcrypto.Sign)
	if err != nil && strings.HasPrefix(err.Error(), "key not exist ") {
		newsignaddr, err := ks.NewKeyWithDefaultPassword(groupId, localcrypto.Sign)
		if err == nil && newsignaddr != "" {
			err = nodeoptions.SetSignKeyMap(groupId, newsignaddr)
			if err != nil {
				return "", errors.New(fmt.Sprintf("save key map %s err: %s", newsignaddr, err.Error()))
			}
			hexkey, err = ks.GetEncodedPubkey(groupId, localcrypto.Sign)
		} else {
			return "", errors.New("create new group key err:" + err.Error())
		}
	}
	return hexkey, nil
}

func initEncodeKey(groupId string, bks localcrypto.Keystore) (string, error) {
	userEncryptKey, err := bks.GetEncodedPubkey(groupId, localcrypto.Encrypt)
	if err != nil {
		if strings.HasPrefix(err.Error(), "key not exist ") {
			userEncryptKey, err = bks.NewKeyWithDefaultPassword(groupId, localcrypto.Encrypt)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return userEncryptKey, nil
}
