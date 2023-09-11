package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rumsystem/quorum/internal/pkg/utils"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type ParseGroupSeedParam struct {
	Seed []byte `param:"seed" validate:"required" example:"seed"`
}

type ParseGroupSeedResult struct {
	GroupId        string                     `json:"groupId"`
	GroupName      string                     `json:"groupName"`
	OwnerPubkey    string                     `json:"ownerPubkey"`
	ProducerPubkey string                     `json:"producerPubkey"`
	SyncType       string                     `json:"syncType"`
	CipherKey      string                     `json:"cipherKey"`
	AppId          string                     `json:"appId"`
	AppName        string                     `json:"appName"`
	ConsensusInfo  *quorumpb.PoaConsensusInfo `json:"consensusInfo"`
	BrewService    *quorumpb.BrewServiceItem  `json:"brewService"`
	SyncService    *quorumpb.SyncServiceItem  `json:"syncService"`
	GenesisBlock   *quorumpb.Block            `json:"genesisBlock"`
	Hash           []byte                     `json:"hash"`
	Signature      []byte                     `json:"sign"`
}

func (h *Handler) ParseGroupSeed(c echo.Context) (err error) {
	cc := c.(*utils.CustomContext)
	params := new(ParseGroupSeedParam)
	if err := cc.BindAndValidate(params); err != nil {
		return err
	}

	seed := &quorumpb.GroupSeed{}
	err = proto.Unmarshal(params.Seed, seed)
	if err != nil {
		return err
	}
	result := &ParseGroupSeedResult{
		GroupId:        seed.GroupId,
		GroupName:      seed.GroupName,
		OwnerPubkey:    seed.OwnerPubkey,
		ProducerPubkey: seed.GenesisBlock.ProducerPubkey,
		SyncType:       seed.SyncType.String(),
		CipherKey:      seed.CipherKey,
		AppId:          seed.AppId,
		AppName:        seed.AppName,
		GenesisBlock:   seed.GenesisBlock,
		Hash:           seed.Hash,
		Signature:      seed.Signature,
	}

	consensus := &quorumpb.PoaConsensusInfo{}
	err = proto.Unmarshal(seed.GenesisBlock.Consensus.Data, consensus)

	if err != nil {
		return err
	}

	result.ConsensusInfo = consensus

	//retrieve services
	for _, serviceItem := range seed.Services {
		if serviceItem.Type == quorumpb.GroupServiceType_SYNC_SERVICE {
			syncService := &quorumpb.SyncServiceItem{}
			err = proto.Unmarshal(serviceItem.Service, syncService)
			if err != nil {
				return err
			}
			result.SyncService = syncService
		} else if serviceItem.Type == quorumpb.GroupServiceType_BREW_SERVICE {
			brewService := &quorumpb.BrewServiceItem{}
			err = proto.Unmarshal(serviceItem.Service, brewService)
			if err != nil {
				return err
			}
			result.BrewService = brewService
		}
	}
	return c.JSON(http.StatusOK, result)
}
