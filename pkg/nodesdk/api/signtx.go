package nodesdkapi

import (
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
		Keyalias string `json:"keyalias" validate:"required"`
		Nonce    uint64 `json:"nonce"`
		To       string `json:"to" validate:"required,eth_addr"` // To eth address, e.g.: 0xab...
		Value    int64  `json:"value"`
		GasLimit uint64 `json:"gas_limit"`
		GasPrice int64  `json:"gas_price"`
		Data     string `json:"data" validate:"required"` // Data hex encode string, include prefix string '0x'
		ChainID  int64  `json:"chain_id" validate:"required"`
	}

	FmtSignTxParam struct {
		Keyalias string         `json:"keyalias" validate:"required"`
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

func loadSignTxParam(param SignTxParam) (*FmtSignTxParam, error) {
	data, err := hexutil.Decode(param.Data)
	if err != nil {
		return nil, err
	}
	fmtParam := FmtSignTxParam{
		Keyalias: param.Keyalias,
		Nonce:    param.Nonce,
		To:       common.HexToAddress(param.To),
		Value:    big.NewInt(param.Value),
		GasLimit: param.GasLimit,
		GasPrice: big.NewInt(param.GasPrice),
		Data:     data,
		ChainID:  big.NewInt(param.ChainID),
	}
	return &fmtParam, nil
}

// @Tags Keystore
// @Summary SignTx
// @Description signature transaction with private key
// @Accept json
// @Produce json
// @Param data body handlers.SignTxParam true "tx param"
// @Success 200 {object} handlers.SignTxResult
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

	ks := nodesdkctx.GetKeyStore()
	dirks, ok := ks.(*localcrypto.DirKeyStore)
	if !ok {
		output[ERROR_INFO] = "Open keystore failed"
		return c.JSON(http.StatusBadRequest, output)
	}
	data, err := dirks.SignTxByKeyAlias(
		param.Keyalias,
		param.Nonce,
		param.To,
		param.Value,
		param.GasLimit,
		param.GasPrice,
		param.Data,
		param.ChainID,
	)
	if err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	result := SignTxResult{Data: data}
	return c.JSON(http.StatusOK, result)
}
