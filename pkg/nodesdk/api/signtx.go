package nodesdkapi

import (
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	localcrypto "github.com/rumsystem/keystore/pkg/crypto"
	nodesdkctx "github.com/rumsystem/quorum/pkg/nodesdk/nodesdkctx"
)

type (
	SignTxParam struct {
		Keyalias string `json:"keyalias"`
		Keyname  string `json:"keyname"`
		Nonce    uint64 `json:"nonce"`
		To       string `json:"to" validate:"required,eth_addr"` // To eth address, e.g.: 0xab...
		Value    string `json:"value"`                           // Value string, the number prefix determines the actual base
		GasLimit uint64 `json:"gas_limit"`
		GasPrice string `json:"gas_price"`                    // GasPrice string, the number prefix determines the actual base
		Data     string `json:"data" validate:"required"`     // Data hex encode string, include prefix string "0x"
		ChainID  string `json:"chain_id" validate:"required"` // ChainID string, the number prefix determines the actual base
	}

	FmtSignTxParam struct {
		Keyalias string         `json:"keyalias"`
		Keyname  string         `json:"keyname"`
		Nonce    uint64         `json:"nonce" validate:"required"`
		To       common.Address `json:"to" validate:"required"` // To eth address, e.g.: 0xab...
		Value    *big.Int       `json:"value" validate:"required"`
		GasLimit uint64         `json:"gas_limit" validate:"required"`
		GasPrice *big.Int       `json:"gas_price" validate:"required"`
		Data     []byte         `json:"data" validate:"required"` // Data base64 encoded string
		ChainID  *big.Int       `json:"chain_id" validate:"required"`
	}

	SignTxResult struct {
		Data string `json:"data" validate:"required"`
	}
)

func strToBigInt(s string) (*big.Int, error) {
	n, ok := big.NewInt(0).SetString(s, 0)
	if !ok {
		return nil, fmt.Errorf("cast %s to big.Int failed", s)
	}

	return n, nil
}

func loadSignTxParam(param SignTxParam) (*FmtSignTxParam, error) {
	data, err := hexutil.Decode(param.Data)
	if err != nil {
		return nil, err
	}
	value, err := strToBigInt(param.Value)
	if err != nil {
		return nil, err
	}
	gasPrice, err := strToBigInt(param.GasPrice)
	if err != nil {
		return nil, err
	}
	chainID, err := strToBigInt(param.ChainID)
	if err != nil {
		return nil, err
	}

	fmtParam := FmtSignTxParam{
		Keyalias: param.Keyalias,
		Keyname:  param.Keyname,
		Nonce:    param.Nonce,
		To:       common.HexToAddress(param.To),
		Value:    value,
		GasLimit: param.GasLimit,
		GasPrice: gasPrice,
		Data:     data,
		ChainID:  chainID,
	}
	return &fmtParam, nil
}

// @Tags Keystore
// @Summary SignTx
// @Description signature transaction with key name or key alias
// @Accept json
// @Produce json
// @Param data body SignTxParam true "tx param"
// @Success 200 {object} SignTxResult
// @Router /nodesdk_api/v1/keystore/signtx [post]
func (h *NodeSDKHandler) SignTx(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	var input SignTxParam

	if err = c.Bind(&input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}
	if err = validate.Struct(input); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	param, err := loadSignTxParam(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	if param.Keyalias == "" && param.Keyname == "" {
		return c.JSON(http.StatusBadRequest, errors.New("both key alias and key name are empty"))
	}

	ks := nodesdkctx.GetKeyStore()
	dirks, ok := ks.(*localcrypto.DirKeyStore)
	if !ok {
		output[ERROR_INFO] = "Open keystore failed"
		return c.JSON(http.StatusBadRequest, output)
	}
	var data string
	if param.Keyalias != "" {
		data, err = dirks.SignTxByKeyAlias(
			param.Keyalias,
			param.Nonce,
			param.To,
			param.Value,
			param.GasLimit,
			param.GasPrice,
			param.Data,
			param.ChainID,
		)
	} else {
		data, err = dirks.SignTxByKeyName(
			param.Keyname,
			param.Nonce,
			param.To,
			param.Value,
			param.GasLimit,
			param.GasPrice,
			param.Data,
			param.ChainID,
		)
	}
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	result := SignTxResult{Data: data}
	return c.JSON(http.StatusOK, result)
}
