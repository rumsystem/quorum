package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/rumsystem/quorum/internal/pkg/handlers"
	"github.com/rumsystem/quorum/testnode"
)

func announceProducer(api string, payload handlers.AnnounceParam) (*handlers.AnnounceResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := string(payloadByte)
	_, resp, err := testnode.RequestAPI(api, "/api/v1/group/announce", "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.AnnounceResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		e := fmt.Errorf("validate.Struct failed: %s, result: %+v", err, result)
		return nil, e
	}

	action := strings.ToUpper(payload.Action)
	if result.Action != action {
		e := fmt.Errorf("result.Action != %s, result: %+v", action, result)
		return nil, e
	}

	_type := strings.ToUpper(payload.Type)
	if _type == "producer" {
		_type = "AS_PRODUCER"
	} else if _type == "user" {
		_type = "AS_USER"
	}

	if payload.Type == "producer" {
		if result.Type != "AS_PRODUCER" {
			e := fmt.Errorf("result.Type != AS_PRODUCER, result: %+v", result)
			return nil, e
		}
	} else if payload.Type == "user" {
		if result.Type != "AS_USER" {
			e := fmt.Errorf("result.Type != AS_USER, result: %+v", result)
			return nil, e
		}
	}

	return &result, nil
}

func getAnnouncedProducers(api string, groupID string) ([]handlers.AnnouncedProducerListItem, error) {
	urlSuffix := fmt.Sprintf("/api/v1/group/%s/announced/producers", groupID)

	_, resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result []handlers.AnnouncedProducerListItem
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	for _, item := range result {
		validate := validator.New()
		if err := validate.Struct(item); err != nil {
			e := fmt.Errorf("validate.Struct failed: %s, result: %+v", err, item)
			return nil, e
		}
	}

	return result, nil
}

func getProducers(api string, groupID string) ([]handlers.ProducerListItem, error) {
	urlSuffix := fmt.Sprintf("/api/v1/group/%s/producers", groupID)
	_, resp, err := testnode.RequestAPI(api, urlSuffix, "GET", "")
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var producers []handlers.ProducerListItem
	if err := json.Unmarshal(resp, &producers); err != nil {
		e := fmt.Errorf("json.Unmarshal failed: %s, response: %s", err, resp)
		return nil, e
	}

	for _, producer := range producers {
		validate := validator.New()
		if err := validate.Struct(producer); err != nil {
			e := fmt.Errorf("validate.Struct failed: %s, producer: %+v", err, producer)
			return nil, e
		}
	}

	return producers, nil
}

// add producer by group owner
func addProducer(api string, payload handlers.GrpProducerParam) (*handlers.GrpProducerResult, error) {
	payloadByte, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	payloadStr := string(payloadByte)

	_, resp, err := testnode.RequestAPI(api, "/api/v1/group/producer", "POST", payloadStr)
	if err != nil {
		return nil, err
	}

	if err := getResponseError(resp); err != nil {
		return nil, err
	}

	var result handlers.GrpProducerResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(result); err != nil {
		e := fmt.Errorf("validate.Struct failed: %s, result: %+v", err, result)
		return nil, e
	}

	return &result, nil
}

func TestAnnounceProducer(t *testing.T) {
	// create group
	createGroupParam := handlers.CreateGroupParam{
		GroupName:      "test-announce-prd",
		ConsensusType:  "poa",
		EncryptionType: "public",
		AppKey:         "default",
	}
	group, err := createGroup(peerapi, createGroupParam)
	if err != nil {
		t.Fatalf("createGroup failed: %s", err)
	}

	// peer2 join group
	joinGroupParam := handlers.GroupSeed{
		GenesisBlock:   group.GenesisBlock,
		GroupId:        group.GroupId,
		GroupName:      group.GroupName,
		OwnerPubkey:    group.OwnerPubkey,
		ConsensusType:  group.ConsensusType,
		EncryptionType: group.EncryptionType,
		CipherKey:      group.CipherKey,
		AppKey:         group.AppKey,
		Signature:      group.Signature,
	}
	if _, err := joinGroup(peerapi2, joinGroupParam); err != nil {
		t.Fatalf("joinGroup failed: %s, payload: %+v", err, joinGroupParam)
	}

	// peer2 announce as producer
	announcePayload := handlers.AnnounceParam{
		GroupId: group.GroupId,
		Action:  "add",
		Type:    "producer",
		Memo:    "producer p1, realiable and cheap, online 24hr",
	}
	announceResult, err := announceProducer(peerapi2, announcePayload)
	if err != nil {
		t.Fatalf("announceProducer failed: %s, payload: %+v", err, announcePayload)
	}

	// group owner should be able to get announced producers
	time.Sleep(time.Second * 30)
	announcedProducers, err := getAnnouncedProducers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getAnnouncedProducers failed: %s", err)
	}

	// check if the producer is in the announced producers list
	if announcedProducers == nil || len(announcedProducers) != 1 {
		t.Fatalf("announcedProducers should only have one item, not %d", len(announcedProducers))
	}

	if announceResult.AnnouncedSignPubkey != announcedProducers[0].AnnouncedPubkey {
		t.Fatalf("announceResult.AnnouncedSignPubkey != announcedProducers[0].SignPubkey, announceResult: %+v, announcedProducers: %+v", announceResult, announcedProducers)
	}

	if announcedProducers[0].Result != "ANNOUNCED" {
		t.Fatalf("announcedProducers[0].Result != ANNOUNCED, announcedProducers: %+v", announcedProducers)
	}

	// group owner approve producer
	peer2PublicKey := announcedProducers[0].AnnouncedPubkey
	producerParam := handlers.GrpProducerParam{
		Action:         "add",
		ProducerPubkey: peer2PublicKey,
		GroupId:        group.GroupId,
		Memo:           "owner-approve",
	}
	if _, err := addProducer(peerapi, producerParam); err != nil {
		t.Fatalf("addProducer failed: %s, payload: %+v", err, producerParam)
	}

	// check approved status
	time.Sleep(time.Second * 15)
	approvedProducers, err := getAnnouncedProducers(peerapi, group.GroupId)
	if err != nil {
		t.Errorf("getAnnouncedProducers failed: %s", err)
	}
	if approvedProducers == nil || len(approvedProducers) != 1 {
		t.Errorf("approvedProducers should only have one item.")
	}
	if approvedProducers[0].AnnouncedPubkey != peer2PublicKey {
		t.Errorf("approvedProducers[0].AnnouncedPubkey != peer2PublicKey, approvedProducers: %+v", approvedProducers)
	}
	if approvedProducers[0].Result != "APPROVED" {
		t.Errorf("approvedProducers[0].Result != APPROVED, approvedProducers: %+v", approvedProducers)
	}
	if approvedProducers[0].Action != "ADD" {
		t.Errorf("approvedProducers[0].Action != ADD, approvedProducers: %+v", approvedProducers)
	}

	// get producers
	producers, err := getProducers(peerapi, group.GroupId)
	if err != nil {
		t.Fatalf("getProducers failed: %s", err)
	}

	// check if the producer is in the producers list
	foundProducer := false
	for _, producer := range producers {
		if producer.ProducerPubkey == peer2PublicKey {
			foundProducer = true
			break
		}
	}
	if !foundProducer {
		t.Fatalf("producer should be in the producers list")
	}
}
