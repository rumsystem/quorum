package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/pkg/chainapi/handlers"
	"github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/encoding/protojson"
)

func getTrx(api string, groupID string, trxID string) (*pb.Trx, error) {
	urlSuffix := fmt.Sprintf("/api/v1/trx/%s/%s", groupID, trxID)
	_, resp, err := requestAPI(api, urlSuffix, "GET", nil, nil, nil, true)
	if err != nil {
		return nil, err
	}

	var result pb.Trx
	if err := protojson.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestGetTrx(t *testing.T) {
	t.Parallel()

	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-get-trx",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Errorf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	// post to group
	content := fmt.Sprintf("%s hello world", RandString(4))
	name := fmt.Sprintf("%s post to group testing", RandString(4))
	postGroupParam := PostGroupParam{
		Data: map[string]interface{}{
			"type":    "Note",
			"content": content,
			"name":    name,
		},
		GroupID: group.GroupId,
	}

	postResult, err := postToGroup(peerapi, postGroupParam)
	if err != nil {
		t.Errorf("postToGroup failed: %s, payload: %+v", err, postGroupParam)
	}
	if postResult.TrxId == "" {
		t.Errorf("postToGroup failed: TrxId is empty")
	}

	// FIXME: wait for trx to be confirmed
	time.Sleep(time.Second * 25)

	// get trx
	trx, err := getTrx(peerapi, group.GroupId, postResult.TrxId)
	if err != nil {
		t.Errorf("getTrx failed: %s, groupID: %s, trxID: %s", err, group.GroupId, postResult.TrxId)
	}

	if trx.TrxId != postResult.TrxId {
		t.Errorf("getTrx failed: TrxId is not equal, expected: %s, actual: %s", postResult.TrxId, trx.TrxId)
	}
}
