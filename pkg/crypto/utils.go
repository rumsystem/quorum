package crypto

import (
	"errors"

	"strings"

	"github.com/rumsystem/quorum/internal/pkg/options"
)

func InitSignKeyWithKeyName(keyname string, nodeoptions *options.NodeOptions) (string, error) {
	_, err := ks.GetEncodedPubkey(keyname, Sign)
	if err == nil {
		return "", errors.New("key already exist")
	}

	if !strings.HasPrefix(err.Error(), "key not exist") {
		return "", err
	}
	//create create key
	newsignaddr, err := ks.NewKeyWithDefaultPassword(keyname, Sign)
	if err != nil {
		return "", errors.New("create new key failed, err:" + err.Error())
	}

	if newsignaddr == "" {
		return "", errors.New("create new key failed, addr is empty")
	}

	err = nodeoptions.SetSignKeyMap(keyname, newsignaddr)
	if err != nil {
		return "", errors.New("save to keymap with addr " + newsignaddr + " failed, err:" + err.Error())
	}
	b64key, err := ks.GetEncodedPubkey(keyname, Sign)
	if err != nil {
		return "", errors.New("get new key failed, err:" + err.Error())
	}

	return b64key, nil

}
