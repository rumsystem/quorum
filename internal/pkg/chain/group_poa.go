package chain

import (
	"errors"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"google.golang.org/protobuf/proto"
)

type GroupPoa struct {
	//Group Item
	Item *quorumpb.GroupItem

	Status GroupStatus

	//Trx
	TrxPool map[string]*quorumpb.Trx // all trx

	//Produce routine timer
	ProduceTimer *time.Timer
	ProduceDone  chan bool

	//Ask next block ticker
	AskNextTimer     *time.Timer
	AskNextTimerDone chan bool

	receivedBlock int
}

func (grp *GroupPoa) Init(item *quorumpb.GroupItem) {
	grp.Item = item
	grp.initProduce()
}

//initial trx pool
func (grp *GroupPoa) initProduce() {
	group_log.Infof("initial trx pool")
	grp.TrxPool = make(map[string]*quorumpb.Trx)
}

//teardown group
func (grp *GroupPoa) Teardown() {
	if grp.Status == GROUP_DIRTY {
		grp.stopWaitBlock()
	}
}

//Start sync group
func (grp *GroupPoa) StartSync() error {
	group_log.Infof("Group %s start syncing", grp.Item.GroupId)

	pubkey, _ := GetChainCtx().GetPubKey()
	if pubkey == grp.Item.OwnerPubKey {
		group_log.Infof("I am the owner, no need to ask new block")
		grp.Status = GROUP_CLEAN
	} else {
		grp.Status = GROUP_DIRTY
		grp.askNextBlock()
		grp.waitBlock()
	}
	return nil
}

//Stop sync group
func (grp *GroupPoa) StopSync() error {
	group_log.Infof("Group stop sync")
	grp.stopWaitBlock()
	grp.Status = GROUP_CLEAN
	return nil
}

func (grp *GroupPoa) askNextBlock() {
	group_log.Infof("Ask NEXT_BLOCK")
	//send ask next block msg out
	//set received block in this round to 0
	grp.receivedBlock = 0

	topBlock, err := grp.GetTopBlock()
	if err != nil {
		group_log.Fatalf(err.Error())
	}

	var reqBlockItem quorumpb.ReqBlock
	reqBlockItem.BlockId = topBlock.BlockId
	reqBlockItem.GroupId = grp.Item.GroupId
	reqBlockItem.UserId = GetChainCtx().PeerId.Pretty()

	bItemBytes, err := proto.Marshal(&reqBlockItem)
	if err != nil {
		group_log.Warningf(err.Error())
		return
	}

	//send ask next block trx out
	trx, err := CreateTrx(quorumpb.TrxType_REQ_BLOCK, grp.Item.GroupId, bItemBytes)
	if err != nil {
		group_log.Warningf(err.Error())
		return
	}

	err = grp.sendTrxPackage(trx)
	if err != nil {
		group_log.Warningf(err.Error())
	}

	// grp.sendTrxPackage(trx)
}

//ask next block
func (grp *GroupPoa) waitBlock() {
	grp.AskNextTimer = (time.NewTimer)(time.Duration(WAIT_BLOCK_TIME_S) * time.Second)
	grp.AskNextTimerDone = make(chan bool)
	go func() {
		for {
			select {
			case <-grp.AskNextTimerDone:
				group_log.Infof("Wait stopped by signal")
				return
			case <-grp.AskNextTimer.C:
				group_log.Infof("Wait done, no new BLOCK_IN_TRX received in 10 sec")
				if grp.receivedBlock == 0 {
					group_log.Infof("Nothing received in this round, start new round(ASK_NEXT_BLOCK)")
					grp.askNextBlock()
					grp.waitBlock()
				} else {
					group_log.Infof("ask next done")
				}
			}
		}
	}()
}

func (grp *GroupPoa) stopWaitBlock() {
	grp.AskNextTimer.Stop()
	grp.AskNextTimerDone <- true
}

func (grp *GroupPoa) HandleReqBlockResp(reqBlockResp *quorumpb.ReqBlockResp) error {
	var newBlock quorumpb.Block
	if err := proto.Unmarshal(reqBlockResp.Block, &newBlock); err != nil {
		return err
	}

	//count block received
	grp.receivedBlock++

	if reqBlockResp.Provider == grp.Item.OwnerPubKey { //block from group owner
		group_log.Infof("Block form group owner, accept it")
		if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_ON_TOP {
			chain_log.Infof("Block from Group owner: BLOCK_ON_TOP, stop sync, set Group Status to GROUP_READY")
			grp.StopSync()
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_IN_TRX {
			chain_log.Infof("Block from Group owner: BLOCK_IN_TRX, add it if don't have it yet")
			topBlock, _ := grp.GetTopBlock()
			if valid, _ := IsBlockValid(&newBlock, topBlock); valid {
				chain_log.Infof("block is valid, add it")
				//add block to db
				GetDbMgr().AddBlock(&newBlock)
				//update group block seq map
				grp.AddBlock(&newBlock)
			} else {
				//already have the block from someone else in this group
				chain_log.Infof("Block from GroupOwner, already got it from other group member")
			}
			//ask next block
			grp.stopWaitBlock()
			grp.askNextBlock()
			grp.waitBlock()
		}
	} else { //block from group memeber
		if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_IN_TRX {
			chain_log.Infof("Block from Group member: BLOCK_IN_TRX, add it if don't have it yet")
			topBlock, _ := grp.GetTopBlock()
			if valid, _ := IsBlockValid(&newBlock, topBlock); valid {
				chain_log.Infof("block is valid, add it")
				//add block to db
				GetDbMgr().AddBlock(&newBlock)
				//update group block seq map
				grp.AddBlock(&newBlock)
				//ask next block
				grp.stopWaitBlock()
				grp.askNextBlock()
				grp.waitBlock()

			} else {
				chain_log.Infof("Block from group member, already got it or invalid block")
			}
		} else if reqBlockResp.Result == quorumpb.ReqBlkResult_BLOCK_ON_TOP {
			chain_log.Infof("Block from Group member: BLOCK_ON_TOP, do nothing, wait till timeout")
		}
	}
	return nil
}

func (grp *GroupPoa) GetTopBlock() (*quorumpb.Block, error) {
	return GetDbMgr().GetBlock(grp.Item.LatestBlockId)
}

func (grp *GroupPoa) GetBlockId(blockNum int64) (string, error) {
	return GetDbMgr().GetBlkId(blockNum, grp.Item.GroupId)
}

func (grp *GroupPoa) CreateGrp(item *quorumpb.GroupItem) error {
	grp.Init(item)

	err := GetDbMgr().AddBlock(item.GenesisBlock)
	if err != nil {
		return err
	}

	err = GetDbMgr().UpdBlkSeq(item.GenesisBlock)
	if err != nil {
		return err
	}

	return GetDbMgr().AddGroup(grp.Item)
}

func (grp *GroupPoa) DelGrp() error {
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(GetChainCtx().PublicKey)
	if err != nil {
		return err
	}

	if grp.Item.OwnerPubKey != p2pcrypto.ConfigEncodeKey(pubkeybytes) {
		err := errors.New("You can not 'delete' group created by others, use 'leave' instead")
		return err
	}

	return GetDbMgr().RmGroup(grp.Item)
}

func (grp *GroupPoa) LeaveGrp() error {
	pubkeybytes, err := p2pcrypto.MarshalPublicKey(GetChainCtx().PublicKey)
	if err != nil {
		return err
	}

	if grp.Item.OwnerPubKey == p2pcrypto.ConfigEncodeKey(pubkeybytes) {
		err := errors.New("Group creator can not leave the group they created, use 'delete' instead")
		return err
	}

	//TODO
	//should clean up all group related data
	return GetDbMgr().RmGroup(grp.Item)
}

//Add trx to trx pool, prepare for produce block
func (grp *GroupPoa) AddTrx(trx *quorumpb.Trx) {

	pubkey, _ := GetChainCtx().GetPubKey()

	//only group owner collect trxs
	if pubkey == grp.Item.OwnerPubKey {
		group_log.Infof("Add Trx")
		grp.TrxPool[trx.TrxId] = trx
		if len(grp.TrxPool) == 1 {
			grp.LaunchProduce()
		}
	} else {
		group_log.Infof("receive trx, wait block from group owner")
	}
}

func (grp *GroupPoa) Post(content *quorumpb.Object) (string, error) {
	group_log.Infof("Post")
	encodedcontent, err := proto.Marshal(content)
	if err != nil {
		return "", err
	}

	group_log.Infof("Create Trx")
	trx, err := CreateTrx(quorumpb.TrxType_POST, grp.Item.GroupId, encodedcontent)
	err = grp.sendTrxPackage(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	grp.AddTrx(trx)
	return trx.TrxId, nil
}

func (grp *GroupPoa) UpdAuth(item *quorumpb.BlockListItem) (string, error) {
	group_log.Infof("Update Auth")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}

	group_log.Infof("Create Trx")
	trx, err := CreateTrx(quorumpb.TrxType_AUTH, grp.Item.GroupId, encodedcontent)
	err = grp.sendTrxPackage(trx)
	if err != nil {
		return "INVALID_TRX", err
	}
	grp.AddTrx(trx)
	return trx.TrxId, nil
}

func (grp *GroupPoa) LaunchProduce() {
	group_log.Infof("Start produce routine")
	go grp.startProduceBlock()
}

func (grp *GroupPoa) startProduceBlock() {
	group_log.Infof("start produce block...")
	grp.ProduceTimer = time.NewTimer(5 * time.Second)
	defer grp.ProduceTimer.Stop()

	for {
		select {
		case t := <-grp.ProduceTimer.C:
			group_log.Infof("Producer wait done at " + t.UTC().String())
			grp.produceBlock()
			grp.initProduce()
		}
	}
}

func (grp *GroupPoa) AddBlock(block *quorumpb.Block) error {
	group_log.Infof("add block")
	topBlock, err := grp.GetTopBlock()

	if err != nil {
		return err
	}

	valid, err := IsBlockValid(block, topBlock)
	if !valid {
		return err
	}

	//check if block produced by group owner
	if block.ProducerPubKey != grp.Item.OwnerPubKey {
		group_log.Info("block not produced by owner, Reject block")
		err = grp.sendNewBlockResp(block, quorumpb.NewBlockRespResult_BLOCK_REJECTED)
	} else {
		group_log.Infof("try apply block")
		err = grp.applyBlock(block)
		if err != nil {
			return err
		}

		if grp.Status != GROUP_DIRTY {
			group_log.Info("Accept block")
			err = grp.sendNewBlockResp(block, quorumpb.NewBlockRespResult_BLOCK_ACCEPTED)

			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (grp *GroupPoa) UpdateNewBlockResp(trx *quorumpb.Trx) error {
	group_log.Infof("UpdateNewBlockResp called")

	/*
		newBlockResp := &quorumpb.NewBLockResp{}
		if err := proto.Unmarshal(trx.Data, newBlockResp); err != nil {
			return err
		}

		if newBlockResp.Result == quorumpb.NewBlockRespResult_BLOCK_ACCEPTED {
			//do nothing
		}
	*/

	return nil
}

func (grp *GroupPoa) applyBlock(block *quorumpb.Block) error {
	group_log.Infof("apply block to group")

	//Save block to local db
	err := GetDbMgr().AddBlock(block)
	if err != nil {
		return err
	}

	//Update block sequence table
	err = GetDbMgr().UpdBlkSeq(block)
	if err != nil {
		return err
	}

	//apply all trx inside block
	for _, trx := range block.Trxs {
		//Save Trx to local Db
		GetDbMgr().AddTrx(trx)
		switch trx.Type {
		case quorumpb.TrxType_POST:
			group_log.Infof("Apply POST trx")
			GetDbMgr().AddPost(trx)
		case quorumpb.TrxType_AUTH:
			group_log.Infof("Apply AUTH trx")
			GetDbMgr().UpdateBlkListItem(trx)
		default:
			group_log.Infof("Unsupported msgType %s", trx.Type)
		}
	}

	//update group info
	grp.Item.LatestBlockNum = block.BlockNum
	grp.Item.LatestBlockId = block.BlockId
	grp.Item.LastUpdate = time.Now().UnixNano()

	//update local db
	dbMgr.UpdGroup(grp.Item)

	return nil
}

func (grp *GroupPoa) produceBlock() {
	group_log.Infof("produce block")

	//get top block
	topBlock, err := grp.GetTopBlock()
	if err != nil {
		group_log.Infof(err.Error())
	}

	//package all trx
	group_log.Infof("package %d trxs", len(grp.TrxPool))
	trxs := make([]*quorumpb.Trx, 0, len(grp.TrxPool))
	for _, value := range grp.TrxPool {
		trxs = append(trxs, value)
	}

	//create block
	newBlock, err := CreateBlock(topBlock, trxs)
	if err != nil {
		group_log.Infof(err.Error())
	}

	//send block via group channel
	grp.sendBlkPackage(newBlock)
	if err != nil {
		group_log.Infof(err.Error())
	}
}

func (grp *GroupPoa) sendTrxPackage(trx *quorumpb.Trx) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_TRX
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	return GetChainCtx().GroupTopicPublish(trx.GroupId, pkgBytes)
}

func (grp *GroupPoa) sendNewBlockResp(block *quorumpb.Block, result quorumpb.NewBlockRespResult) error {
	var newBlockRespItem quorumpb.NewBLockResp
	newBlockRespItem.BlockId = block.BlockId
	newBlockRespItem.GroupId = grp.Item.GroupId
	newBlockRespItem.ProducerId = block.ProducerId
	newBlockRespItem.Receiver = GetChainCtx().PeerId.Pretty()
	newBlockRespItem.Result = result

	bItemBytes, err := proto.Marshal(&newBlockRespItem)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}
	trx, err := CreateTrx(quorumpb.TrxType_NEW_BLOCK_RESP, grp.Item.GroupId, bItemBytes)
	if err != nil {
		group_log.Warningf(err.Error())
		return err
	}

	return grp.sendTrxPackage(trx)
}

func (grp *GroupPoa) sendBlkPackage(blk *quorumpb.Block) error {
	var pkg *quorumpb.Package
	pkg = &quorumpb.Package{}

	pbBytes, err := proto.Marshal(blk)
	if err != nil {
		return err
	}

	pkg.Type = quorumpb.PackageType_BLOCK
	pkg.Data = pbBytes

	pkgBytes, err := proto.Marshal(pkg)
	if err != nil {
		return err
	}

	err = GetChainCtx().GroupTopicPublish(blk.GroupId, pkgBytes)

	if err != nil {
		return err
	}

	return nil
}

func (grp *GroupPoa) UpdateChallenge(trx *quorumpb.Trx) error {
	//should not happened here
	return nil
}
