package nodesdkapi

import (
	"net/http"
	"os"

	guuid "github.com/google/uuid"

	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/options"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type CreateNewKeyWithAliasParams struct {
	Alias string `json:"alias" validate:"required"`
	Type  string `json:"type"  validate:"required,oneof=encrypt sign"`
}

type CreateNewKeyWithAliasResult struct {
	Alias   string
	Keyname string
	KeyType string
	Pubkey  string
}

func (h *NodeSDKHandler) CreateNewKeyWithAlias() echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := c.(*utils.CustomContext)
		params := new(CreateNewKeyWithAliasParams)
		if err := cc.BindAndValidate(params); err != nil {
			return err
		}

		var keytype localcrypto.KeyType
		switch params.Type {
		case "sign":
			keytype = localcrypto.Sign
		case "encrypt":
			keytype = localcrypto.Encrypt
		}

		nodeoptions := options.GetNodeOptions()
		ks := nodesdkctx.GetKeyStore()
		dirks, ok := ks.(*localcrypto.DirKeyStore)
		if !ok {
			return rumerrors.NewBadRequestError("Open keystore failed")
		}

		keyname := dirks.AliasToKeyname(params.Alias)
		if keyname != "" {
			return rumerrors.NewBadRequestError("Existed alias")
		}

		password := os.Getenv("RUM_KSPASSWD")
		keyname = guuid.New().String()

		newsignaddr, err := dirks.NewKey(keyname, keytype, password)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if err := nodeoptions.SetSignKeyMap(keyname, newsignaddr); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		if err := dirks.NewAlias(params.Alias, keyname, password); err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		pubkey, err := dirks.GetEncodedPubkey(keyname, keytype)
		if err != nil {
			return rumerrors.NewBadRequestError(err)
		}

		result := &CreateNewKeyWithAliasResult{
			Alias:   params.Alias,
			Keyname: keyname,
			KeyType: params.Type,
			Pubkey:  pubkey,
		}
		return c.JSON(http.StatusOK, result)
	}
}
