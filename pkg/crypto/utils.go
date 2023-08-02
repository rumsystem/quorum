package crypto

import (
	"errors"

	"strings"

	"github.com/rumsystem/quorum/internal/pkg/options"
)

func InitSignKeyWithKeyName(keyname string, nodeoptions *options.NodeOptions) (string, error) {
	b64key, err := ks.GetEncodedPubkey(keyname, Sign)
	if err != nil && strings.HasPrefix(err.Error(), "key not exist ") {
		newsignaddr, err := ks.NewKeyWithDefaultPassword(keyname, Sign)
		if err == nil && newsignaddr != "" {
			err = nodeoptions.SetSignKeyMap(keyname, newsignaddr)
			if err != nil {
				return "", errors.New("save key map " + newsignaddr + " err:" + err.Error())
			}
			b64key, err = ks.GetEncodedPubkey(keyname, Sign)
			if err != nil {
				return "", errors.New("create new group key err:" + err.Error())
			}
		} else {
			return "", errors.New("create new group key err:" + err.Error())
		}
	}
	return b64key, nil
}
