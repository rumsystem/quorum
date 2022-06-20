package nodesdkapi

import (
	"fmt"
	"net/http"
	"os"

	guuid "github.com/google/uuid"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/options"
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
		var err error
		output := make(map[string]string)

		validate := validator.New()
		params := new(CreateNewKeyWithAliasParams)

		if err = c.Bind(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		if err = validate.Struct(params); err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
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
			output[ERROR_INFO] = "Open keystore failed"
			return c.JSON(http.StatusBadRequest, output)
		}

		keyname := dirks.AliasToKeyname(params.Alias)
		if keyname != "" {
			output[ERROR_INFO] = "Existed alias"
			return c.JSON(http.StatusBadRequest, output)
		}

		password := os.Getenv("RUM_KSPASSWD")
		keyname = guuid.New().String()

		newsignaddr, err := dirks.NewKey(keyname, keytype, password)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		err = nodeoptions.SetSignKeyMap(keyname, newsignaddr)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		err = dirks.NewAlias(params.Alias, keyname, password)
		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		pubkey, err := dirks.GetEncodedPubkey(keyname, keytype)
		if err != nil {
			fmt.Println(err.Error())
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}

		result := &CreateNewKeyWithAliasResult{Alias: params.Alias, Keyname: keyname, KeyType: params.Type, Pubkey: pubkey}
		return c.JSON(http.StatusOK, result)
	}
}
