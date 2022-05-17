package api

import (
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

type AnnouncedGroupUserList struct {
	Data []*handlers.AnnouncedUserListItem `json:"data"`
}

func GetAnnouncedGroupUsers(groupId string) (*AnnouncedGroupUserList, error) {
	wasmCtx := quorumContext.GetWASMContext()
	res, err := handlers.GetAnnouncedGroupUsers(wasmCtx.GetChainStorage(), groupId)
	if err != nil {
		return nil, err
	}
	return &AnnouncedGroupUserList{res}, nil
}
