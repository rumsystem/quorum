package chain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	iface "github.com/rumsystem/quorum/internal/pkg/chaindataciface"
	"github.com/rumsystem/quorum/internal/pkg/conn"
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
	nodeName string
	//group            *Group
	GroupId          string
	Group            *Group
	AskNextTimer     *time.Timer
	AskNextTimerDone chan bool
	Status           int8
	retryCount       int8
	statusBeforeFail int8
	responses        map[string]*quorumpb.ReqBlockResp
	blockReceived    map[string]string
	cdnIface         iface.ChainDataHandlerIface
	syncNetworkType  conn.P2pNetworkType
	locker           sync.RWMutex
}

func (syncer *Syncer) Init(group *Group, cdnIface iface.ChainDataHandlerIface) {
	syncer_log.Debugf("<%s> Init called", group.Item.GroupId)
	syncer.Status = IDLE
	syncer.Group = group
	syncer.GroupId = group.Item.GroupId
	syncer.retryCount = 0
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	syncer.blockReceived = make(map[string]string)
	syncer.cdnIface = cdnIface
	syncer.syncNetworkType = conn.PubSub
	syncer_log.Infof("<%s> syncer initialed", syncer.GroupId)
}

// sync block "forward"
func (syncer *Syncer) SyncForward(block *quorumpb.Block) error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> SyncForward called", syncer.GroupId)

	//no need to sync for producers(owner)
	if syncer.Group.Item.OwnerPubKey == syncer.Group.Item.UserSignPubkey {
		if len(syncer.Group.ChainCtx.ProducerPool) == 1 {
			syncer_log.Debugf("<%s> group owner, no registed producer, no need to sync", syncer.GroupId)
			return nil
		} else {
			syncer_log.Debugf("<%s> owner, has registed producer, start sync missing block", syncer.GroupId)
		}
	} else if _, ok := syncer.Group.ChainCtx.ProducerPool[syncer.Group.Item.UserSignPubkey]; ok {
		syncer_log.Debugf("<%s> producer, no need to sync forward (sync backward when new block produced and found missing block(s)", syncer.GroupId)
		return nil
	} else if syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD {
		return errors.New("already in SYNCING")
	}

	syncer_log.Debugf("<%s> try sync forward from block <%s>", syncer.GroupId, block.BlockId)
	syncer.blockReceived = make(map[string]string)
	syncer.Status = SYNCING_FORWARD
	err := syncer.askNextBlock(block)
	if err != nil {
		syncer_log.Debugf("<%s> askNextBlock <%s> return err: %s", syncer.GroupId, block.BlockId, err)
	}
	syncer.waitBlock(block)
	return nil
}

//Sync block "backward"
func (syncer *Syncer) SyncBackward(block *quorumpb.Block) error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> SyncBackward called", syncer.GroupId)

	//if I am the owner
	if syncer.Group.Item.OwnerPubKey == syncer.Group.Item.UserSignPubkey &&
		len(syncer.Group.ChainCtx.ProducerPool) == 1 {
		syncer_log.Warningf("<%s> owner, no producer exist, no need to sync, SOMETHING WRONG HAPPENED", syncer.GroupId)
		return nil
	}

	if syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD {
		return errors.New("already in SYNCING")
	}

	syncer.blockReceived = make(map[string]string)
	syncer.Status = SYNCING_BACKWARD
	syncer.askPreviousBlock(block)
	syncer.waitBlock(block)
	return nil
}

func (syncer *Syncer) StopSync() error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> StopSync called", syncer.GroupId)
	syncer.stopWaitBlock()
	syncer.Status = IDLE
	syncer_log.Debugf("<%s> sync stopped", syncer.GroupId)
	return nil
}

func (syncer *Syncer) ContinueSync(block *quorumpb.Block) error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> ContinueSync called", syncer.GroupId)
	syncer.stopWaitBlock()
	if syncer.Status == SYNCING_FORWARD {
		err := syncer.askNextBlock(block)
		if err != nil {
			syncer_log.Debugf("<%s> askNextBlock <%s> return err: %s", syncer.GroupId, block.BlockId, err)
		}
		syncer.waitBlock(block)

	} else if syncer.Status == SYNCING_BACKWARD {
		err := syncer.askPreviousBlock(block)
		if err != nil {
			syncer_log.Debugf("<%s> askPreviousBlock <%s> return err: %s", syncer.GroupId, block.BlockId, err)
		}
		syncer.waitBlock(block)
	} else if syncer.Status == SYNC_FAILED {
		syncer_log.Debugf("<%s> Sync faileld, should manually start sync", syncer.GroupId)
	} else {
		//idle
		syncer_log.Debugf("<%s> syncer idle, can not continue", syncer.GroupId)
	}

	return nil
}

func (syncer *Syncer) AddBlockSynced(resp *quorumpb.ReqBlockResp, block *quorumpb.Block) error {

	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> AddBlockSynced called", syncer.GroupId)
	if !(syncer.Status == SYNCING_FORWARD || syncer.Status == SYNCING_BACKWARD) {
		syncer_log.Warningf("<%s> Not in syncing, ignore block", syncer.GroupId)
		return nil
	}

	//block in trx
	syncer_log.Debugf("<%s> synced block incoming, provider <%s>", syncer.GroupId, resp.ProviderPubkey)
	syncer.responses[resp.ProviderPubkey] = resp

	if resp.Result == quorumpb.ReqBlkResult_BLOCK_NOT_FOUND {
		syncer_log.Debugf("<%s> receive BLOCK_NOT_FOUND response, do nothing(wait for timeout)", syncer.GroupId)
		return nil
	}

	if _, blockReceived := syncer.blockReceived[resp.BlockId]; blockReceived {
		syncer_log.Debugf("<%s> Block with Id <%s> already received", syncer.GroupId, resp.BlockId)
		return nil
	}

	_, producer := syncer.Group.ChainCtx.ProducerPool[syncer.Group.Item.UserSignPubkey]

	if syncer.Status == SYNCING_FORWARD {
		if producer {
			syncer_log.Debugf("<%s> SYNCING_FORWARD, PRODUCER ADD BLOCK", syncer.GroupId)
			err := syncer.Group.ChainCtx.Consensus.Producer().AddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		} else {
			syncer_log.Debugf("<%s> SYNCING_FORWARD, USER ADD BLOCK", syncer.GroupId)
			err := syncer.Group.ChainCtx.Consensus.User().AddBlock(block)
			if err != nil {
				syncer_log.Infof(err.Error())
			}
		}

		syncer_log.Debugf("<%s> SYNCING_FORWARD, CONTINUE", syncer.GroupId)
		syncer.blockReceived[resp.BlockId] = resp.ProviderPubkey
		syncer.ContinueSync(block)
	} else { //sync backward
		var err error
		if producer {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD, PRODUCER ADD BLOCK", syncer.GroupId)
			err = syncer.Group.ChainCtx.Consensus.Producer().AddBlock(block)
		} else {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD, USER ADD BLOCK", syncer.GroupId)
			err = syncer.Group.ChainCtx.Consensus.User().AddBlock(block)
		}

		syncer.blockReceived[resp.BlockId] = resp.ProviderPubkey
		if err != nil {
			syncer_log.Debugf(err.Error())
			if err.Error() == "PARENT_NOT_EXIST" {
				syncer_log.Debugf("<%s> SYNCING_BACKWARD, CONTINUE", syncer.GroupId)
				syncer.ContinueSync(block)
			}
		} else {
			syncer_log.Debugf("<%s> SYNCING_BACKWARD err is nil", syncer.GroupId)
		}
	}

	return nil
}

func (syncer *Syncer) askNextBlock(block *quorumpb.Block) error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()
	syncer_log.Debugf("<%s> askNextBlock called, block id: %s", syncer.GroupId, block.BlockId)
	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block forward msg out
	trx, err := syncer.Group.ChainCtx.GetTrxFactory().GetReqBlockForwardTrx(block)
	if err != nil {
		return err
	}

	connMgr, err := conn.GetConn().GetConnMgr(syncer.GroupId)
	if err != nil {
		return err
	}

	if syncer.syncNetworkType == conn.PubSub {
		return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
	} else {
		return connMgr.SendTrxRex(trx, "")
	}

	return nil
}

func (syncer *Syncer) askPreviousBlock(block *quorumpb.Block) error {
	//syncer.locker.Lock()
	//defer syncer.locker.Unlock()

	syncer_log.Debugf("<%s> askPreviousBlock called", syncer.GroupId)

	//reset received response
	syncer.responses = make(map[string]*quorumpb.ReqBlockResp)
	//send ask block backward msg out
	trx, err := syncer.Group.ChainCtx.GetTrxFactory().GetReqBlockBackwardTrx(block)
	if err != nil {
		return err
	}

	connMgr, err := conn.GetConn().GetConnMgr(syncer.GroupId)
	if err != nil {
		return err
	}

	if syncer.syncNetworkType == conn.PubSub {
		return connMgr.SendTrxPubsub(trx, conn.ProducerChannel)
	} else {
		return connMgr.SendTrxRex(trx, "")
	}

	return nil
}

//wait block coming
func (syncer *Syncer) waitBlock(block *quorumpb.Block) {

	syncer_log.Debugf("<%s> waitBlock called", syncer.GroupId)
	syncer.AskNextTimer = (time.NewTimer)(time.Duration(WAIT_BLOCK_TIME_S) * time.Second)
	syncer.AskNextTimerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-syncer.AskNextTimerDone:
				syncer_log.Debugf("<%s> wait stopped by signal", syncer.GroupId)
				return
			case <-syncer.AskNextTimer.C:
				syncer_log.Debugf("<%s> wait done", syncer.GroupId)
				if len(syncer.responses) == 0 {
					syncer.retryCount++
					//switch network type and retry
					if syncer.syncNetworkType == conn.PubSub {
						syncer.syncNetworkType = conn.RumExchange
					} else {
						syncer.syncNetworkType = conn.PubSub
					}
					syncer_log.Debugf("<%s> nothing received in this round, start new round (retry time: <%d>), set p2p network type to: [%s]", syncer.GroupId, syncer.retryCount, syncer.syncNetworkType)
					if syncer.retryCount == int8(RETRY_LIMIT) {
						syncer_log.Debugf("<%s> reach retry limit <%d>, SYNC FAILED, check network connection", syncer.GroupId, RETRY_LIMIT)
						//save syncer status
						syncer.statusBeforeFail = syncer.Status
						syncer.Status = SYNC_FAILED
						return
					}
					if syncer.Status == SYNCING_FORWARD {
						err := syncer.askNextBlock(block)
						if err != nil {
							syncer_log.Debugf("<%s> askNextBlock <%s> return err: %s", syncer.GroupId, block.BlockId, err)
						}
						syncer.waitBlock(block)
					} else if syncer.Status == SYNCING_BACKWARD {
						syncer.askPreviousBlock(block)
						syncer.waitBlock(block)
					}
				} else { // all BLOCK_NOT_FOUND
					syncer_log.Debugf("<%s> received <%d> BLOCK_NOT_FOUND resp, sync done, set to IDLE", syncer.GroupId, len(syncer.responses))
					syncer.Status = IDLE
				}
			}
		}
	}()
}

func (syncer *Syncer) stopWaitBlock() {
	syncer_log.Debugf("<%s> stopWaitBlock called", syncer.GroupId)
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
	syncer_log.Debugf("<%s> ShowChainStruct called", syncer.GroupId)
	genesisblkid := syncer.Group.ChainCtx.group.Item.GenesisBlock.BlockId

	chainstruct, err := syncer.GetBlockToGenesis(syncer.Group.Item.HighestBlockId, genesisblkid)
	if err != nil {
		syncer_log.Errorf("<%s> ChainStruct genesis <%s> err <%s>", syncer.GroupId, genesisblkid, err)
	} else {
		syncer_log.Debugf("<%s> ChainStruct genesis <%s> struct: <%s>", syncer.GroupId, genesisblkid, chainstruct)
	}
}
