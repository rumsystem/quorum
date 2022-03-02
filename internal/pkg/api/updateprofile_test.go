package api

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

type Image struct {
	Name      string `json:"name,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Content   []byte `json:"content,omitempty"`
	Url       string `json:"url,omitempty"`
}

type Payment struct {
	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`
}

type Person struct {
	Name   string     `json:"name,omitempty"`
	Image  *Image     `json:"image,omitempty"`
	Wallet []*Payment `json:"wallet,omitempty"`
}

type updateProfileParam struct {
	Type   string     `json:"type" validate:"required,oneof=Update"`
	Person Person     `json:"person" validate:"required"`
	Target PostTarget `json:"target" validate:"required"`
}

var avatar = []byte{
	137, 80, 78, 71, 13, 10, 26, 10, 0, 0, 0, 13, 73, 72, 68, 82, 0, 0, 0, 1, 0,
	0, 0, 1, 1, 3, 0, 0, 0, 37, 219, 86, 202, 0, 0, 0, 3, 80, 76, 84, 69, 0, 0,
	0, 167, 122, 61, 218, 0, 0, 0, 1, 116, 82, 78, 83, 0, 64, 230, 216, 102, 0,
	0, 0, 10, 73, 68, 65, 84, 8, 215, 99, 96, 0, 0, 0, 2, 0, 1, 226, 33, 188,
	51, 0, 0, 0, 0, 73, 69, 78, 68, 174, 66, 96, 130,
}

func updateProfile(api string, payload updateProfileParam) (*UpdateProfileResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	payloadStr := string(payloadBytes)

	urlSuffix := "/api/v1/group/profile"
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result UpdateProfileResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		return nil, err
	}

	return &result, nil
}

func TestUpdateAvatar(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "update-avatar",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	// update user avatar
	updateProfileParam := updateProfileParam{
		Type: "Update",
		Person: Person{
			Image: &Image{
				MediaType: "image/png",
				Content:   avatar,
			},
		},
		Target: PostTarget{
			Type: "Group",
			ID:   group.GroupId,
		},
	}
	if _, err := updateProfile(peerapi, updateProfileParam); err != nil {
		t.Fatalf("updateProfile failed: %s, payload: %+v", err, updateProfileParam)
	}
}

func TestUpdateNickname(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "update-nickname",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s, payload: %+v", err, createGroupParam)
	}

	// update user avatar
	updateProfileParam := updateProfileParam{
		Type: "Update",
		Person: Person{
			Name: "just for testing",
		},
		Target: PostTarget{
			Type: "Group",
			ID:   group.GroupId,
		},
	}
	if _, err := updateProfile(peerapi, updateProfileParam); err != nil {
		t.Fatalf("updateProfile failed: %s, payload: %+v", err, updateProfileParam)
	}
}
