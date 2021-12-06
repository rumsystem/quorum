package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/rumsystem/quorum/internal/pkg/handlers"
)

func TestGetGroups(t *testing.T) {
	appKey := "default"
	consensusType := "poa"
	encryptionTypes := []string{"public", "private"}

	for _, encryptionType := range encryptionTypes {
		groupName := fmt.Sprintf("%s-%d", encryptionType, time.Now().Unix())
		payload := handlers.CreateGroupParam{
			AppKey:         appKey,
			ConsensusType:  consensusType,
			EncryptionType: encryptionType,
			GroupName:      groupName,
		}

		if _, err := createGroup(peerapi, payload); err != nil {
			t.Errorf("create group failed: %s", err)
		}

		groups, err := getGroups(peerapi)
		if err != nil {
			t.Errorf("getGroups failed: %s", err)
		}

		if groups.GroupInfos == nil {
			t.Error("it should least one group, but groups is null")
		}
	}
}
