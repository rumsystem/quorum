package api

import (
	rumerrors "github.com/rumsystem/quorum/internal/pkg/errors"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func LeaveGroup(groupId string) (*handlers.LeaveGroupResult, error) {
	if groupId == "" {
		return nil, rumerrors.ErrInvalidGroupID
	}
	wasmCtx := quorumContext.GetWASMContext()
	params := handlers.LeaveGroupParam{GroupId: groupId}
	return handlers.LeaveGroup(&params, wasmCtx.AppDb)
}
