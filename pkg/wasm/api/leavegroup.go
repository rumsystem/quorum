package api

import (
	"errors"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
	quorumContext "github.com/rumsystem/quorum/pkg/wasm/context"
)

func LeaveGroup(groupId string) (*handlers.LeaveGroupResult, error) {
	if groupId == "" {
		return nil, errors.New("empty group id")
	}
	wasmCtx := quorumContext.GetWASMContext()
	params := handlers.LeaveGroupParam{GroupId: groupId}
	return handlers.LeaveGroup(&params, wasmCtx.AppDb)
}
