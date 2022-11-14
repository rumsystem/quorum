package appapi

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	chain "github.com/rumsystem/quorum/internal/pkg/chainsdk/core"
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/internal/pkg/storage/def"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type GroupContentObjectItem struct {
	TrxId     string
	Publisher string
	Content   proto.Message
	TypeUrl   string
	TimeStamp int64
}

type SenderList struct {
	Senders []string
}

// @Tags Apps
// @Summary GetGroupContents
// @Description Get contents in a group
// @Produce json
// @Param group_id path string  true "Group Id"
// @Param num query string false "the count of returns results"
// @Param reverse query boolean false "reverse = true will return results by most recently"
// @Param starttrx query string false "returns results from this trxid, but exclude it"
// @Param includestarttrx query string false "include the start trx"
// @Param nonce query int false "the nonce of trx, the default value is the latest"
// @Param data body SenderList true "SenderList"
// @Success 200 {array} GroupContentObjectItem
// @Router /app/api/v1/group/{group_id}/content [post]
func (h *Handler) ContentByPeers(c echo.Context) (err error) {
	groupid := c.Param("group_id")
	num, _ := strconv.Atoi(c.QueryParam("num"))
	nonce, _ := strconv.ParseInt(c.QueryParam("nonce"), 10, 64)
	starttrx := c.QueryParam("starttrx")
	if num == 0 {
		num = 20
	}
	reverse := false
	if c.QueryParam("reverse") == "true" {
		reverse = true
	}
	includestarttrx := false
	if c.QueryParam("includestarttrx") == "true" {
		includestarttrx = true
	}
	senderlist := &SenderList{}
	if err = c.Bind(&senderlist); err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	trxids, err := h.Appdb.GetGroupContentBySenders(groupid, senderlist.Senders, starttrx, nonce, num, reverse, includestarttrx)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	groupmgr := chain.GetGroupMgr()
	groupitem, err := groupmgr.GetGroupItem(groupid)
	if err != nil {
		return rumerrors.NewBadRequestError(err)
	}

	ctnobjList := []*GroupContentObjectItem{}
	for _, trxid := range trxids {
		trx, _, err := h.Trxdb.GetTrx(groupid, trxid.TrxId, def.Chain, h.NodeName)
		if err != nil {
			c.Logger().Errorf("GetTrx Err: %s", err)
			continue
		}

		//decrypt trx data
		if trx.Type == quorumpb.TrxType_POST && groupitem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by age for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(groupid, trx.Data)
			if err != nil {
				//can't decrypt, replace it
				trx.Data = []byte("")
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

		ctnobj, typeurl, errum := quorumpb.BytesToMessage(trx.TrxId, trx.Data)
		if errum != nil {
			c.Logger().Errorf("Unmarshal trx.Data %s Err: %s", trx.TrxId, errum)
		}
		pk, _ := localcrypto.Libp2pPubkeyToEthBase64(trx.SenderPubkey)
		ctnobjitem := &GroupContentObjectItem{TrxId: trx.TrxId, Publisher: pk, Content: ctnobj, TimeStamp: trx.TimeStamp, TypeUrl: typeurl}
		ctnobjList = append(ctnobjList, ctnobjitem)
	}
	return c.JSON(http.StatusOK, ctnobjList)
}
