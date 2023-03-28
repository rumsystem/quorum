package appapi

import (
	"encoding/hex"
	"net/http"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
)

// @Tags Apps
// @Summary GetGroupContents
// @Description Get contents in a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param params query handlers.GetGroupCtnPrarms false "get group contents params"
// @Param data body SenderList true "SenderList"
// @Success 200 {array} GroupContentObjectItem
// @Router /app/api/v1/group/{group_id}/content [get]
func (h *Handler) ContentByPeers(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	var params handlers.GetGroupCtnPrarms
	if err := cc.BindAndValidate(&params); err != nil {
		return err
	}
	if params.Num <= 0 {
		params.Num = 20
	}

	trxids, err := h.Appdb.GetGroupContentBySenders(params.GroupId, params.Senders, params.StartTrx, params.Num, params.Reverse, params.IncludeStartTrx)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	groupmgr := chain.GetGroupMgr()
	groupitem, err := groupmgr.GetGroupItem(params.GroupId)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	res := []*quorumpb.Trx{}
	for _, trxid := range trxids {
		trx, err := h.Trxdb.GetTrx(params.GroupId, trxid, def.Chain, h.NodeName)
		if err != nil {
			logger.Errorf("GetTrx groupid: %s trxid: %s failed: %s", params.GroupId, trxid, err)
			continue
		}
		if trx.TrxId == "" && len(trx.Data) == 0 {
			logger.Warnf("GetTrx groupid: %s trxid: %s return empty trx, skip ...", params.GroupId, trxid)
			continue
		}

		//decrypt trx data
		if trx.Type == quorumpb.TrxType_POST && groupitem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by age for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(params.GroupId, trx.Data)
			if err != nil {
				//can't decrypt, replace it
				trx.Data = nil
				logger.Warnf("can not decrypt trx.Data for groupid: %s trxid: %s failed: %s", params.GroupId, trxid, err)
			} else {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}
		} else {
			//decode trx data
			ciperKey, err := hex.DecodeString(groupitem.CipherKey)
			if err != nil {
				return err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return err
			}
			trx.Data = decryptData
		}

		res = append(res, trx)
	}
	return c.JSON(http.StatusOK, res)
}
