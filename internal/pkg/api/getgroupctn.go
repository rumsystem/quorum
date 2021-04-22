package api

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"github.com/go-playground/validator/v10"
	"github.com/golang/glog"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
)

type GetGroupCtnParams struct {
	GroupId string `from:"group_id" json:"group_id" validate:"required"`
}

func (h *Handler) GetGroupCtn(c echo.Context) (err error) {
	output := make(map[string]string)

	validate := validator.New()
	params := new(GetGroupCtnParams)
	if err = c.Bind(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if err = validate.Struct(params); err != nil {
		output[ERROR_INFO] = err.Error()
		return c.JSON(http.StatusBadRequest, output)
	}

	if group, ok := chain.GetChainCtx().Groups[params.GroupId]; ok {
		var ctnList []*chain.GroupContentItem
		err = group.Db.ContentDb.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchSize = 10
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := item.Key()
				err := item.Value(func(v []byte) error {
					i := int64(binary.LittleEndian.Uint64(k))
					s := strconv.Itoa(int(i))
					var contentitem *chain.GroupContentItem
					ctnerr := json.Unmarshal(v, &contentitem)
					if ctnerr == nil {
						ctnList = append(ctnList, contentitem)
					} else {
						glog.Errorf("unknown data format: %s", v)
					}
					output[s] = string(v)
					return nil
				})

				if err != nil {
					return err
				}
			}

			return nil
		})

		if err != nil {
			output[ERROR_INFO] = err.Error()
			return c.JSON(http.StatusBadRequest, output)
		}
		return c.JSON(http.StatusOK, ctnList)
	} else {
		output[ERROR_INFO] = fmt.Sprintf("Group %s not exist", params.GroupId)
		return c.JSON(http.StatusBadRequest, output)
	}
}
