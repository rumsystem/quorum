package api

import (
	"net/http"

	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
)

type BlkUserListItem struct {
	GroupId     string
	UserId      string
	OwnerPubkey string
	OwnerSign   string
	Memo        string
	TimeStamp   int64
}

type BlkUserList struct {
	BlkList []*BlkUserListItem `json:"blocked"`
}

func (h *Handler) GetBlockedUsrList(c echo.Context) (err error) {
	var result []*BlkUserListItem

	blkList, err := chain.GetDbMgr().GetBlkListItems()

	for _, blkItem := range blkList {

		var item *BlkUserListItem
		item = &BlkUserListItem{}
		item.GroupId = blkItem.GroupId
		item.UserId = blkItem.UserId
		item.OwnerPubkey = blkItem.OwnerPubkey
		item.OwnerSign = blkItem.OwnerSign
		item.Memo = blkItem.Memo
		item.TimeStamp = blkItem.TimeStamp
		result = append(result, item)
	}

	return c.JSON(http.StatusOK, &BlkUserList{result})
}
