package chain

import (
	"errors"
	"fmt"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
)

var syncer_log = logging.Logger("syncer")

var WAIT_BLOCK_TIME_S = 10 //wait time period
var RETRY_LIMIT = 5        //retry times

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
}

func (syncer *Syncer) Init(grp *Group, trxMgr *TrxMgr) {
	syncer.Status = IDLE
	syncer.group = grp
	syncer.trxMgr = trxMgr
	syncer.retryCount = 0
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
}

// sync block "forward"
func (syncer *Syncer) SyncForward(block *quorumpb.Block) error {
	syncer_log.Infof("Group %s start syncing forward on node %s", syncer.group.Item.GroupId, syncer.nodeName)

	//no need to sync for producers(owner)
	if syncer.group.Item.OwnerPubKey == syncer.group.Item.UserSignPubkey {
		if len(syncer.group.ChainCtx.ProducerPool) == 1 {
			syncer_log.Infof("I am the owner, no producer exist, no need to sync")
			return errors.New("I am the owner, no producer exist, no need to sync")
		} else {
			syncer_log.Infof("I am the owner, producer exist, sync missing block")
		}
	} else if _, ok := syncer.group.ChainCtx.ProducerPool[syncer.group.Item.UserSignPubkey]; ok {
		syncer_log.Infof("I am a producer, no need to sync forward (sync backward when new block produced and found missing block(s)")
		return errors.New("I am a producer, no need to sync forward (sync backward when new block produced and found missing block(s)")
	} else if syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD {
		return errors.New("already in SYNCING")
	}

	syncer_log.Infof("try sync forward from block %s", block.BlockId)
	syncer.Status = SYNCING_FORWARD
	syncer.askNextBlock(block)
	syncer.waitBlock(block)
	return nil
}

//Sync block "backward"
func (syncer *Syncer) SyncBackward(block *quorumpb.Block) error {
	syncer_log.Infof("Group %s start syncing backward on node %s", syncer.group.Item.GroupId, syncer.nodeName)
	syncer_log.Infof("try sync backward from block %s", block.BlockId)

	//if I am the owner
	if syncer.group.Item.OwnerPubKey == syncer.group.Item.UserSignPubkey &&
		len(syncer.group.ChainCtx.ProducerPool) == 1 {
		syncer_log.Warningf("I am the owner, no producer exist, no need to sync, SOMETHING WRONG HAPPENED")
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
	syncer_log.Infof("Group stop sync")
	syncer.stopWaitBlock()
	syncer.Status = IDLE
	syncer_log.Infof("Group stop done")
	return nil
}

func (syncer *Syncer) ContinueSync(block *quorumpb.Block) error {
	syncer_log.Infof("ContinueSync called")
	syncer.stopWaitBlock()
	if syncer.Status == SYNCING_FORWARD {
		syncer.askNextBlock(block)
		syncer.waitBlock(block)
	} else if syncer.Status == SYNCING_BACKWARD {
		syncer.askPreviousBlock(block)
		syncer.waitBlock(block)
	} else if syncer.Status == SYNC_FAILED {
		syncer_log.Infof("TBD, Sync faileld, should manually start sync")
	} else {
		//idle
		syncer_log.Warningf("Syncer idle, can not continue")
	}

	return nil
}

func (syncer *Syncer) AddBlockSynced(resp *quorumpb.ReqBlockResp, block *quorumpb.Block) error {

	if !(syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD) {
		syncer_log.Warningf("Not in syncing, ignore block")
		return nil
	}

	//block in trx
	syncer_log.Infof("Add response from %s", resp.ProviderPubkey)
	syncer.responses[resp.ProviderPubkey] = resp

	if resp.Result == quorumpb.ReqBlkResult_BLOCK_NOT_FOUND {
		syncer_log.Infof("receive BLOCK_NOT_FOUND response, do nothing(wait for timeout)")
		return nil
	}

	_, producer := syncer.group.ChainCtx.ProducerPool[syncer.group.Item.UserSignPubkey]

	if syncer.Status == SYNCING_FORWARD {
		if producer {
			syncer_log.Infof("SYNCING_FORWARD, PRODUCER ADD BLOCK")
			err := syncer.group.ChainCtx.ProducerAddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		} else {
			syncer_log.Infof("SYNCING_FORWARD, USER ADD BLOCK")
			err := syncer.group.ChainCtx.UserAddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		}
		syncer_log.Infof("SYNCING_FORWARD, CONTINUE")
		syncer.ContinueSync(block)
	} else { //sync backward
		var err error
		if producer {
			syncer_log.Infof("SYNCING_BACKWARD, PRODUCER ADD BLOCK")
			err = syncer.group.ChainCtx.ProducerAddBlock(block)
		} else {
			syncer_log.Infof("SYNCING_BACKWARD, USER ADD BLOCK")
			err = syncer.group.ChainCtx.UserAddBlock(block)
		}
		if err.Error() == "PARENT_NOT_EXIST" {
			syncer_log.Infof("SYNCING_BACKWARD, CONTINUE")
			syncer.ContinueSync(block)
		} else if err != nil {
			syncer_log.Infof(err.Error())
		}
	}

	return nil

}

func (syncer *Syncer) askNextBlock(block *quorumpb.Block) {
	syncer_log.Infof("askNextBlock called")

	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block forward msg out
	syncer.trxMgr.SendReqBlockForward(block)
}

func (syncer *Syncer) askPreviousBlock(block *quorumpb.Block) {
	syncer_log.Infof("askPreviousBlock called")
	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block backward msg out
	syncer.trxMgr.SendReqBlockBackward(block)
}

//wait block coming
func (syncer *Syncer) waitBlock(block *quorumpb.Block) {
	syncer_log.Infof("Start waiting block")
	syncer.AskNextTimer = (time.NewTimer)(time.Duration(WAIT_BLOCK_TIME_S) * time.Second)
	syncer.AskNextTimerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-syncer.AskNextTimerDone:
				syncer_log.Infof("Wait stopped by signal")
				return
			case <-syncer.AskNextTimer.C:
				syncer_log.Infof("Wait done")
				if len(syncer.responses) == 0 {
					syncer.retryCount++
					syncer_log.Infof("Nothing received in this round, start new round (retry time: %d)", syncer.retryCount)
					if syncer.retryCount == int8(RETRY_LIMIT) {
						syncer_log.Warnf("Reach retry limit %d, SYNC FAILED, check network connection", RETRY_LIMIT)
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
					syncer.ShowChainStruct()
				} else { // all BLOCK_NOT_FOUND
					syncer_log.Infof("received %d BLOCK_NOT_FOUND resp, sync done, set to IDLE", len(syncer.responses))
					syncer.Status = IDLE
				}
			}
		}
	}()
}

func (syncer *Syncer) stopWaitBlock() {
	syncer.AskNextTimer.Stop()
	syncer.AskNextTimerDone <- true
}

func (syncer *Syncer) GetBlockToGenesis(blockid string, genesisblkid string) (string, error) {
	blk, err := GetDbMgr().GetBlock(blockid, false, syncer.nodeName)
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
	syncer_log.Infof("ShowChainStruct called: %s", syncer.nodeName)
	genesisblkid := syncer.group.ChainCtx.group.Item.GenesisBlock.BlockId
	for _, blockId := range syncer.group.ChainCtx.group.Item.HighestBlockId {
		chainstruct, err := syncer.GetBlockToGenesis(blockId, genesisblkid)
		if err != nil {
			syncer_log.Errorf("%s ChainStruct genesis %s err %s", syncer.nodeName, genesisblkid, err)
		} else {
			syncer_log.Debugf("%s ChainStruct genesis %s struct: %s", syncer.nodeName, genesisblkid, chainstruct)
		}
	}
}
