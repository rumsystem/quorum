package chain

import (
	"errors"
	"fmt"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
)

var syncer_log = logging.Logger("syncer")

var WAIT_BLOCK_TIME_S = 10 //wait time period
var RETRY_LIMIT = 30       //retry times

//syncer status
const (
	SYNCING_FORWARD  = 0
	SYNCING_BACKWARD = 1
	SYNC_FAILED      = 2
	IDLE             = 3
)

type Syncer struct {
	nodeName         string
	group            *Group
	trxMgr           *TrxMgr
	AskNextTimer     *time.Timer
	AskNextTimerDone chan bool
	Status           int8
	retryCount       int8
	statusBeforeFail int8
	responses        map[string]*quorumpb.ReqBlockResp
	groupId          string
}

func (syncer *Syncer) Init(grp *Group, trxMgr *TrxMgr) {
	syncer_log.Debug("Init called")
	syncer.Status = IDLE
	syncer.group = grp
	syncer.trxMgr = trxMgr
	syncer.retryCount = 0
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	syncer.groupId = grp.Item.GroupId
	syncer_log.Infof("<%s> syncer initialed", syncer.groupId)
}

// sync block "forward"
func (syncer *Syncer) SyncForward(block *quorumpb.Block) error {
	syncer_log.Debugf("<%s> SyncForward called", syncer.group.Item.GroupId)

	//no need to sync for producers(owner)
	if syncer.group.Item.OwnerPubKey == syncer.group.Item.UserSignPubkey {
		if len(syncer.group.ChainCtx.ProducerPool) == 1 {
			syncer_log.Debugf("<%s> group owner, no registed producer, no need to sync", syncer.groupId)
			return nil
		} else {
			syncer_log.Debugf("<%s> owner, has registed producer, start sync missing block", syncer.groupId)
		}
	} else if _, ok := syncer.group.ChainCtx.ProducerPool[syncer.group.Item.UserSignPubkey]; ok {
		syncer_log.Debugf("<%s> producer, no need to sync forward (sync backward when new block produced and found missing block(s)", syncer.groupId)
		return nil
	} else if syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD {
		return errors.New("already in SYNCING")
	}

	syncer_log.Debugf("<%s> try sync forward from block <%s>", syncer.groupId, block.BlockId)
	syncer.Status = SYNCING_FORWARD
	syncer.askNextBlock(block)
	syncer.waitBlock(block)
	return nil
}

//Sync block "backward"
func (syncer *Syncer) SyncBackward(block *quorumpb.Block) error {
	syncer_log.Debugf("<%s> SyncBackward called", syncer.group.Item.GroupId)

	//if I am the owner
	if syncer.group.Item.OwnerPubKey == syncer.group.Item.UserSignPubkey &&
		len(syncer.group.ChainCtx.ProducerPool) == 1 {
		syncer_log.Warningf("<%s> owner, no producer exist, no need to sync, SOMETHING WRONG HAPPENED", syncer.groupId)
		return nil
	}

	if syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD {
		return errors.New("already in SYNCING")
	}

	syncer.Status = SYNCING_BACKWARD
	syncer.askPreviousBlock(block)
	syncer.waitBlock(block)
	return nil
}

func (syncer *Syncer) StopSync() error {
	syncer_log.Debugf("<%s> StopSync called", syncer.groupId)
	syncer.stopWaitBlock()
	syncer.Status = IDLE
	syncer_log.Debugf("<%s> sync stopped", syncer.groupId)
	return nil
}

func (syncer *Syncer) ContinueSync(block *quorumpb.Block) error {
	syncer_log.Debugf("<%s> ContinueSync called", syncer.groupId)
	syncer.stopWaitBlock()
	if syncer.Status == SYNCING_FORWARD {
		syncer.askNextBlock(block)
		syncer.waitBlock(block)
	} else if syncer.Status == SYNCING_BACKWARD {
		syncer.askPreviousBlock(block)
		syncer.waitBlock(block)
	} else if syncer.Status == SYNC_FAILED {
		syncer_log.Debugf("<%s> TBD, Sync faileld, should manually start sync", syncer.groupId)
	} else {
		//idle
		syncer_log.Debugf("<%s> syncer idle, can not continue", syncer.groupId)
	}

	return nil
}

func (syncer *Syncer) AddBlockSynced(resp *quorumpb.ReqBlockResp, block *quorumpb.Block) error {
	syncer_log.Debugf("<%s> AddBlockSynced called", syncer.groupId)
	if !(syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD) {
		syncer_log.Warningf("<%s> Not in syncing, ignore block", syncer.groupId)
		return nil
	}

	//block in trx
	syncer_log.Debugf("<%s> synced block incoming, provider <%s>", syncer.groupId, resp.ProviderPubkey)
	syncer.responses[resp.ProviderPubkey] = resp

	if resp.Result == quorumpb.ReqBlkResult_BLOCK_NOT_FOUND {
		syncer_log.Debugf("<%s> receive BLOCK_NOT_FOUND response, do nothing(wait for timeout)", syncer.groupId)
		return nil
	}

	_, producer := syncer.group.ChainCtx.ProducerPool[syncer.group.Item.UserSignPubkey]

	if syncer.Status == SYNCING_FORWARD {
		if producer {
			syncer_log.Debugf("<%s> SYNCING_FORWARD, PRODUCER ADD BLOCK", syncer.groupId)
			err := syncer.group.ChainCtx.Consensus.Producer().AddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		} else {
			syncer_log.Debugf("<%s> SYNCING_FORWARD, USER ADD BLOCK", syncer.groupId)
			err := syncer.group.ChainCtx.Consensus.User().AddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		}
		syncer_log.Debugf("<%s> SYNCING_FORWARD, CONTINUE", syncer.groupId)
		syncer.ContinueSync(block)
	} else { //sync backward
		var err error
		if producer {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD, PRODUCER ADD BLOCK", syncer.groupId)
			err = syncer.group.ChainCtx.Consensus.Producer().AddBlock(block)
		} else {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD, USER ADD BLOCK", syncer.groupId)
			err = syncer.group.ChainCtx.Consensus.User().AddBlock(block)
		}

		if err != nil {
			syncer_log.Debugf(err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				syncer_log.Debugf("<%s> SYNCING_BACKWARD, CONTINUE", syncer.groupId)
				syncer.ContinueSync(block)
			}
		} else {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD err is nil", syncer.groupId)
		}
	}

	return nil
}

func (syncer *Syncer) askNextBlock(block *quorumpb.Block) {
	syncer_log.Debugf("<%s> askNextBlock called", syncer.groupId)

	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block forward msg out
	syncer.trxMgr.SendReqBlockForward(block)
}

func (syncer *Syncer) askPreviousBlock(block *quorumpb.Block) {
	syncer_log.Debugf("<%s> askPreviousBlock called", syncer.groupId)

	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block backward msg out
	syncer.trxMgr.SendReqBlockBackward(block)
}

//wait block coming
func (syncer *Syncer) waitBlock(block *quorumpb.Block) {
	syncer_log.Debugf("<%s> waitBlock called", syncer.groupId)
	syncer.AskNextTimer = (time.NewTimer)(time.Duration(WAIT_BLOCK_TIME_S) * time.Second)
	syncer.AskNextTimerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-syncer.AskNextTimerDone:
				syncer_log.Debugf("<%s> wait stopped by signal", syncer.groupId)
				return
			case <-syncer.AskNextTimer.C:
				syncer_log.Debugf("<%s> wait done", syncer.groupId)
				if len(syncer.responses) == 0 {
					syncer.retryCount++
					syncer_log.Debugf("<%s> nothing received in this round, start new round (retry time: <%d>)", syncer.groupId, syncer.retryCount)
					if syncer.retryCount == int8(RETRY_LIMIT) {
						syncer_log.Debugf("<%s> reach retry limit <%d>, SYNC FAILED, check network connection", syncer.groupId, RETRY_LIMIT)
						//save syncer status
						syncer.statusBeforeFail = syncer.Status
						syncer.Status = SYNC_FAILED
						return
					}
					if syncer.Status == SYNCING_FORWARD {
						syncer.askNextBlock(block)
						syncer.waitBlock(block)
					} else if syncer.Status == SYNCING_BACKWARD {
						syncer.askPreviousBlock(block)
						syncer.waitBlock(block)
					}
					//syncer.ShowChainStruct()
				} else { // all BLOCK_NOT_FOUND
					syncer_log.Debugf("<%s> received <%d> BLOCK_NOT_FOUND resp, sync done, set to IDLE", syncer.groupId, len(syncer.responses))
					syncer.Status = IDLE
				}
			}
		}
	}()
}

func (syncer *Syncer) stopWaitBlock() {
	syncer_log.Debugf("<%s> stopWaitBlock called", syncer.groupId)
	if syncer.AskNextTimer != nil {
		syncer.AskNextTimer.Stop()
		syncer.AskNextTimerDone <- true
	}
}

func (syncer *Syncer) GetBlockToGenesis(blockid string, genesisblkid string) (string, error) {
	blk, err := nodectx.GetDbMgr().GetBlock(blockid, false, syncer.nodeName)
	if err != nil {
		return "", err
	}
	if blk.BlockId == genesisblkid { //ok find the genesis block, return...
		return blk.BlockId, nil
	} else {
		prevblkid, err := syncer.GetBlockToGenesis(blk.PrevBlockId, genesisblkid)
		if err == nil {
			return prevblkid + " <= " + fmt.Sprintf("%s (%d trx)", blk.BlockId, len(blk.Trxs)), nil
		} else {
			return "", err
		}
	}
}

func (syncer *Syncer) ShowChainStruct() {
	syncer_log.Debugf("<%s> ShowChainStruct called", syncer.groupId)
	genesisblkid := syncer.group.ChainCtx.group.Item.GenesisBlock.BlockId

	chainstruct, err := syncer.GetBlockToGenesis(syncer.group.Item.HighestBlockId, genesisblkid)
	if err != nil {
		syncer_log.Errorf("<%s> ChainStruct genesis <%s> err <%s>", syncer.groupId, genesisblkid, err)
	} else {
		syncer_log.Debugf("<%s> ChainStruct genesis <%s> struct: <%s>", syncer.groupId, genesisblkid, chainstruct)
	}
}
