package nodesdkapi

import (
	"net/http"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type KeyItem struct {
	Alias   []string
	Keyname string
	Keytype string
}

type GetAllAliasResult struct {
	Keys []*KeyItem `json:"keys"`
}

func (h *NodeSDKHandler) GetAllAlias() echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error
		output := make(map[string]string)
		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		keys, err := dirks.ListAll()
		if err != nil {
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		var keyitems []*KeyItem
		for _, keyitem := range keys {
			var item *KeyItem
			item = &KeyItem{}
			item.Alias = keyitem.Alias
			item.Keyname = keyitem.Keyname
			if keyitem.Type == localcrypto.Encrypt {
				item.Keytype = "encrypt"
			} else {
				item.Keytype = "sign"
			}
			keyitems = append(keyitems, item)
		}

		result := GetAllAliasResult{keyitems}
		return c.JSON(http.StatusOK, &result)
	}
}
