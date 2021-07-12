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

	WaitTimer *time.Timer
	WaitDone  chan bool

	//Ask next block ticker
	AskNextTicker     *time.Ticker
	AskNextTickerDone chan bool
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
		grp.stopAskNextBlock()
	}
}

//Start sync group
func (grp *GroupPoa) StartSync() error {
	group_log.Infof("Group %s start syncing", grp.Item.GroupId)

	pubkey, _ := getPubKey()
	if pubkey == grp.Item.OwnerPubKey {
		group_log.Infof("I am the owner, no need to ask new block")
		grp.Status = GROUP_CLEAN
	} else {
		grp.Status = GROUP_DIRTY
		grp.startAskNextBlock()
	}
	return nil
}

//Stop sync group
func (grp *GroupPoa) StopSync() error {
	group_log.Infof("Group stop sync")
	grp.Status = GROUP_CLEAN
	grp.stopAskNextBlock()
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

	pubkey, _ := getPubKey()

	//only group owner collect trxs
	if pubkey == grp.Item.OwnerPubKey {
		group_log.Infof("Add Trx")
		grp.TrxPool[trx.TrxId] = trx
		if len(grp.TrxPool) == 1 {
			grp.LaunchProduce()
		}
	}
}

func (grp *GroupPoa) PostBytes(trxtype quorumpb.TrxType, encodedcontent []byte) (string, error) {
	group_log.Infof("Create Trx")
	trx, err := CreateTrx(trxtype, grp.Item.GroupId, encodedcontent)
	err = grp.sendTrxPackage(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	grp.AddTrx(trx)
	return trx.TrxId, nil
}

func (grp *GroupPoa) PostAny(content interface{}) (string, error) {
	group_log.Infof("Post any")
	encodedcontent, err := quorumpb.ContentToBytes(content)
	if err != nil {
		return "", err
	}
	return grp.PostBytes(quorumpb.TrxType_POST, encodedcontent)
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

/*
func (grp *GroupPoa) startWaitBlock() {
	group_log.Infof("start wait block...")
	grp.WaitTimer = time.NewTimer(5 * time.Second)
	defer grp.WaitTimer.Stop()

	for {
		select {
		case t := <-grp.WaitTimer.C:
			group_log.Infof("wait done at " + t.UTC().String())
		case <-grp.WaitDone:
			group_log.Infof("Wait finished by channel")
			return
		}
	}
}
*/

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

//ask next block
func (grp *GroupPoa) startAskNextBlock() {
	grp.AskNextTicker = time.NewTicker(1000 * time.Millisecond)
	grp.AskNextTickerDone = make(chan bool)
	//send ask_next_block every 1 sec till get "on_top response"
	go func() {
		for {
			select {
			case <-grp.AskNextTickerDone:
				group_log.Infof("Ask next block done")
				return
			case t := <-grp.AskNextTicker.C:
				group_log.Infof("Ask NEXT_BLOCK " + t.UTC().String())
				//send ask next block msg out
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
					return
				}
				grp.sendTrxPackage(trx)
			}
		}
	}()
}

func (grp *GroupPoa) stopAskNextBlock() {
	grp.AskNextTicker.Stop()
	grp.AskNextTickerDone <- true
	grp.Status = GROUP_CLEAN
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
