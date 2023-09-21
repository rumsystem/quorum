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
	GroupId        string                       `json:"groupId"`
	GroupName      string                       `json:"groupName"`
	OwnerPubkey    string                       `json:"ownerPubkey"`
	ProducerPubkey string                       `json:"producerPubkey"`
	AuthType       string                       `json:"authType"`
	CipherKey      string                       `json:"cipherKey"`
	AppId          string                       `json:"appId"`
	AppName        string                       `json:"appName"`
	ConsensusInfo  *quorumpb.PoaConsensusInfo   `json:"consensusInfo"`
	ProduceService *quorumpb.ProduceServiceItem `json:"brewService"`
	SyncService    *quorumpb.SyncServiceItem    `json:"syncService"`
	CtnService     *quorumpb.CtnServiceItem     `json:"ctnService"`
	PublishSevice  *quorumpb.PublishServiceItem `json:"postService"`
	GenesisBlock   *quorumpb.Block              `json:"genesisBlock"`
	Hash           []byte                       `json:"hash"`
	Signature      []byte                       `json:"sign"`
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
		AuthType:       seed.AuthType.String(),
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
		if serviceItem.TaskType == quorumpb.GroupTaskType_SYNC {
			syncService := &quorumpb.SyncServiceItem{}
			err = proto.Unmarshal(serviceItem.Data, syncService)
			if err != nil {
				return err
			}
			result.SyncService = syncService
		} else if serviceItem.TaskType == quorumpb.GroupTaskType_PRODUCE {
			produceService := &quorumpb.ProduceServiceItem{}
			err = proto.Unmarshal(serviceItem.Data, produceService)
			if err != nil {
				return err
			}
			result.ProduceService = produceService
		} else if serviceItem.TaskType == quorumpb.GroupTaskType_CTN {
			ctnService := &quorumpb.CtnServiceItem{}
			err = proto.Unmarshal(serviceItem.Data, ctnService)
			if err != nil {
				return err
			}
			result.CtnService = ctnService
		} else if serviceItem.TaskType == quorumpb.GroupTaskType_PUBLISH {
			publishService := &quorumpb.PublishServiceItem{}
			err = proto.Unmarshal(serviceItem.Data, publishService)
			if err != nil {
				return err
			}
			result.PublishSevice = publishService
		}
	}
	return c.JSON(http.StatusOK, result)
}
