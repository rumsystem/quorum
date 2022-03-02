package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

type GetBlockResult struct {
	BlockId        string          `json:"BlockId" validate:"required"`
	GroupId        string          `json:"GroupId" validate:"required"`
	PrevBlockId    string          `json:"PrevBlockId,omitempty"`
	PreviousHash   []byte          `json:"PreviousHash,omitempty"`
	Trxs           []*GetTrxResult `json:"Trxs,omitempty"`
	ProducerPubKey string          `json:"ProducerPubKey" validate:"required"`
	Hash           []byte          `json:"Hash" validate:"required"`
	Signature      []byte          `json:"Signature" validate:"required"`
	TimeStamp      string          `json:"TimeStamp" validate:"required"`
}

func getBlockByID(api, groupID, blockID string) (*GetBlockResult, error) {
	urlSuffix := fmt.Sprintf("/api/v1/block/%s/%s", groupID, blockID)
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result GetBlockResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	if result.BlockId != blockID {
		return nil, fmt.Errorf("block id is not equal, expect: %s, actual: %s", blockID, result.BlockId)
	}
	if result.GroupId != groupID {
		return nil, fmt.Errorf("group id is not equal, expect: %s, actual: %s", groupID, result.GroupId)
	}

	valiate := validator.New()
	if err := valiate.Struct(result); err != nil {
		return nil, err
	}

	groups, err := getGroups(api)
	if err != nil {
		return nil, fmt.Errorf("getGroups failed: %s", err)
	}

	var group *groupInfo
	for _, g := range groups.GroupInfos {
		if g.GroupId == groupID {
			group = g
			break
		}
	}

	if group == nil {
		return nil, fmt.Errorf("group not found, groupID: %s", groupID)
	}

	if group.HighestHeight != 0 {
		if result.Trxs == nil {
			return nil, fmt.Errorf("it should have trxs, but trxs is null, groupID: %s blockID: %s, HighestHeight: %d", group.GroupId, group.HighestBlockId, group.HighestHeight)
		}

		if result.PrevBlockId == "" || result.PreviousHash == nil {
			return nil, fmt.Errorf("it should have prev block id and previous hash, but prev block id or previous hash is null, groupID: %s blockID: %s, HighestHeight: %d", group.GroupId, group.HighestBlockId, group.HighestHeight)
		}
	} else {
		if result.PrevBlockId != "" || result.PreviousHash != nil {
			return nil, fmt.Errorf("it should not have prev block id and previous hash, but prev block id: %s, previous hash: %s, groupID: %s blockID: %s, HighestHeight: %d", result.PrevBlockId, result.PreviousHash, group.GroupId, group.HighestBlockId, group.HighestHeight)
		}
		if result.Trxs != nil {
			return nil, fmt.Errorf("it should not have trxs, but trxs is not null, groupID: %s blockID: %s, HighestHeight: %d", group.GroupId, group.HighestBlockId, group.HighestHeight)
		}
	}

	return &result, nil
}

func TestBlockByID(t *testing.T) {
	appKey := "default"
	consensusType := "poa"
	encryptionType := "public"

	groupName := fmt.Sprintf("%s-%d", encryptionType, time.Now().Unix())
	payload := handlers.CreateGroupParam{
		AppKey:         appKey,
		ConsensusType:  consensusType,
		EncryptionType: encryptionType,
		GroupName:      groupName,
	}

	group, err := createGroup(peerapi, payload)
	if err != nil {
		t.Fatalf("create group failed: %s", err)
	}

	blockID := group.GenesisBlock.BlockId
	if _, err := getBlockByID(peerapi, group.GroupId, blockID); err != nil {
		t.Errorf("getBlockByID failed: %s, groupID: %s blockID: %s", err, group.GroupId, blockID)
	}
}
