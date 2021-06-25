package chain

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"sort"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"google.golang.org/protobuf/proto"

	"github.com/mr-tron/base58"
)

type RoutineStatus int8

const (
	IDLE      = 0
	CHALLENGE = 1
	PRODUCE   = 2
)

type ChallengeGroup struct {
	//Group Item
	Item *quorumpb.GroupItem

	//Status
	Status GroupStatus

	//Trx
	TrxPool map[string]*quorumpb.Trx // all trx

	RStatus RoutineStatus
	//Challenge
	ChallengePool  map[int64]*quorumpb.ChallengeItem
	ChallengeIndex []int64
	IndexPosition  int
	IndexLen       int
	AcceptRecvd    int

	//Produce routine timer
	ChallengeTimer     *time.Timer
	WaitBlockTimer     *time.Timer
	ProduceRoutineDone chan bool

	//Ask next block ticker
	AskNextTicker     *time.Ticker
	AskNextTickerDone chan bool
}

func (grp *ChallengeGroup) Init(item *quorumpb.GroupItem) {
	grp.Item = item
	grp.initProduce()
}

//initial trx pool
func (grp *ChallengeGroup) initProduce() {
	grp.TrxPool = make(map[string]*quorumpb.Trx)
	grp.RStatus = IDLE
	grp.ChallengePool = make(map[int64]*quorumpb.ChallengeItem)
	grp.ChallengeIndex = nil
	grp.IndexPosition = 0
	grp.IndexLen = 0
	grp.AcceptRecvd = 0
}

//teardown group
func (grp *ChallengeGroup) Teardown() {
	if grp.Status == GROUP_DIRTY {
		grp.stopAskNextBlock()
	}
}

//Start sync group
func (grp *ChallengeGroup) StartSync() error {
	group_log.Infof("Group %s start syncing", grp.Item.GroupId)
	grp.Status = GROUP_DIRTY
	grp.startAskNextBlock()
	return nil
}

//Stop sync group
func (grp *ChallengeGroup) StopSync() error {
	group_log.Infof("Group stop sync")
	grp.Status = GROUP_CLEAN
	grp.stopAskNextBlock()
	return nil
}

func (grp *ChallengeGroup) GetTopBlock() (*quorumpb.Block, error) {
	return GetDbMgr().GetBlock(grp.Item.LatestBlockId)
}

func (grp *ChallengeGroup) GetBlockId(blockNum int64) (string, error) {
	return GetDbMgr().GetBlkId(blockNum, grp.Item.GroupId)
}

func (grp *ChallengeGroup) CreateGrp(item *quorumpb.GroupItem) error {
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

func (grp *ChallengeGroup) DelGrp() error {
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

func (grp *ChallengeGroup) LeaveGrp() error {
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
func (grp *ChallengeGroup) AddTrx(trx *quorumpb.Trx) {
	grp.TrxPool[trx.TrxId] = trx
}

func (grp *ChallengeGroup) Post(content *quorumpb.Object) (string, error) {
	encodedcontent, err := proto.Marshal(content)
	if err != nil {
		return "", err
	}

	return grp.LaunchProduce(encodedcontent, quorumpb.TrxType_POST)
}

func (grp *ChallengeGroup) UpdAuth(item *quorumpb.BlockListItem) (string, error) {
	group_log.Infof("Update Auth")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}
	return grp.LaunchProduce(encodedcontent, quorumpb.TrxType_AUTH)

}

//Post to group (by myself)
func (grp *ChallengeGroup) LaunchProduce(content []byte, trxType quorumpb.TrxType) (string, error) {
	group_log.Infof("Launch Produce")
	trx, err := CreateTrx(trxType, grp.Item.GroupId, content)
	err = grp.sendTrxPackage(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	//add my trx to trx pool
	grp.AddTrx(trx)

	//if idle, start a round of challenge
	if grp.RStatus == IDLE {
		var challenge *quorumpb.ChallengeItem
		challenge = &quorumpb.ChallengeItem{}

		challenge.Challenger = GetChainCtx().PeerId.Pretty()
		seed, err := grp.getChallengeSeed(challenge.Challenger)
		challenge.ChallengeSeed = seed

		//add challenge item to challenge pool
		grp.ChallengePool[challenge.ChallengeSeed] = challenge
		grp.ChallengeIndex = append(grp.ChallengeIndex, challenge.ChallengeSeed)

		chItemBytes, err := proto.Marshal(challenge)
		if err != nil {
			return "INVALID_CHALLENGE_TRX", err
		}

		trxChallenge, err := CreateTrx(quorumpb.TrxType_CHALLENGE, grp.Item.GroupId, chItemBytes)
		if err != nil {
			return "INVALID_CHALLENGE_TRX", err
		}

		err = grp.sendTrxPackage(trxChallenge)
		if err != nil {
			return "INVALID_CHALLENGE_TRX", err
		}

		group_log.Infof("==================== Start produce routine ====================")
		go grp.startChallenge()
	}

	return trx.TrxId, err
}

//Start a round of challenge
func (grp *ChallengeGroup) startChallenge() {
	group_log.Infof("startChallenge")
	grp.RStatus = CHALLENGE

	//set timer for 10s
	grp.ChallengeTimer = time.NewTimer(10 * time.Second)
	grp.ProduceRoutineDone = make(chan bool)
	defer grp.ChallengeTimer.Stop()

	for {
		select {
		case t := <-grp.ChallengeTimer.C:
			//sort challenge list
			group_log.Infof("Challenge done, Sort challenge list")
			sort.Slice(grp.ChallengeIndex, func(i, j int) bool {
				return grp.ChallengeIndex[i] < grp.ChallengeIndex[j]
			})
			group_log.Infof("challenge pool index %v", grp.ChallengeIndex)
			group_log.Infof("try produce block " + t.UTC().String())

			//calculate challenge seeds consensus number
			//at least 2/3 node of the challenge list should accept the block unless there is only 1 or 2 node in the list
			grp.IndexLen = len(grp.ChallengeIndex)
			if !(grp.IndexLen == 1 || grp.IndexLen == 2) {
				grp.IndexLen = (int)(grp.IndexLen * 2 / 3)
			}
			go grp.tryProduceBlock()
			return
		case <-grp.ProduceRoutineDone:
			group_log.Infof("In challenge, produce routine stopped by channel")
			grp.finishProduce()
		}
	}
}

func (grp *ChallengeGroup) UpdateChallenge(trx *quorumpb.Trx) error {
	group_log.Infof("Update challenge")

	challenge := &quorumpb.ChallengeItem{}
	if err := proto.Unmarshal(trx.Data, challenge); err != nil {
		return err
	}

	switch grp.RStatus {
	case IDLE:
		group_log.Infof("IDLE, receive challenge item %v", challenge)
		group_log.Infof("create and send my challenge response")

		//initial round of challenge
		var myChallenge *quorumpb.ChallengeItem
		myChallenge = &quorumpb.ChallengeItem{}

		myChallenge.Challenger = GetChainCtx().PeerId.Pretty()
		seed, err := grp.getChallengeSeed(myChallenge.Challenger)
		myChallenge.ChallengeSeed = seed

		chItemBytes, err := proto.Marshal(myChallenge)
		if err != nil {
			return err
		}

		//send challenge response
		trx, err := CreateTrx(quorumpb.TrxType_CHALLENGE, grp.Item.GroupId, chItemBytes)
		err = grp.sendTrxPackage(trx)
		if err != nil {
			return err
		}

		//add my challenge item to pool
		grp.ChallengePool[myChallenge.ChallengeSeed] = myChallenge
		grp.ChallengeIndex = append(grp.ChallengeIndex, myChallenge.ChallengeSeed)

		//add incoming challenge to pool
		grp.ChallengePool[challenge.ChallengeSeed] = challenge
		grp.ChallengeIndex = append(grp.ChallengeIndex, challenge.ChallengeSeed)
		go grp.startChallenge()
	case CHALLENGE:
		group_log.Infof("CHALLENGE, receive challenge item %v", challenge)
		//add incoming challenge to pool
		grp.ChallengePool[challenge.ChallengeSeed] = challenge
		grp.ChallengeIndex = append(grp.ChallengeIndex, challenge.ChallengeSeed)
	case PRODUCE:
		group_log.Infof("in PRODUCE, receive challenge item %v", challenge)
		group_log.Infof("ignore challege item")
	}

	return nil
}

func (grp *ChallengeGroup) tryProduceBlock() {
	group_log.Infof("try produce block...")

	grp.RStatus = PRODUCE
	grp.AcceptRecvd = 0
	index := grp.ChallengeIndex[grp.IndexPosition]

	grp.WaitBlockTimer = time.NewTimer(5 * time.Second)
	defer grp.WaitBlockTimer.Stop()

	//if it is my turn to produce block
	if grp.ChallengePool[index].Challenger == GetChainCtx().PeerId.Pretty() {
		group_log.Infof("My turn to produce block")
		grp.produceBlock()
		group_log.Infof("Start wait")

		if len(grp.ChallengePool) == 1 {
			//only myself, produce block anyway, no need to wait for others
			return
		}

		//start wait 5 seconds
		grp.WaitBlockTimer = time.NewTimer(5 * time.Second)
		for {
			select {
			case t := <-grp.WaitBlockTimer.C:
				group_log.Infof("Producer wait done at " + t.UTC().String())

				//update producer index
				grp.IndexPosition += 1
				if grp.IndexPosition == len(grp.ChallengePool) {
					group_log.Infof("use all challengers and still can not produce block, fail the round and reject all trx")
					grp.stopProduceRoutine()
				}
				group_log.Infof("Don't get 2/3 consensus, update index %d", grp.IndexPosition)
				group_log.Infof("Start next round of waiting")
				grp.tryProduceBlock()
			case <-grp.ProduceRoutineDone:
				group_log.Infof("Produce done")
				grp.finishProduce()
				return
			}
		}
	} else {
		group_log.Infof("Not my turn, wait block incoming")
		for {
			select {
			case t := <-grp.WaitBlockTimer.C:
				group_log.Infof("Wait done at " + t.UTC().String())
				grp.IndexPosition += 1
				if grp.IndexPosition == len(grp.ChallengePool) {
					group_log.Infof("use all challengers and still can not produce block, fail the round and reject all trx")
					grp.stopProduceRoutine()
				}
				group_log.Infof("Don't get the block expected, update index %d ", grp.IndexPosition)
				group_log.Infof("Start next round of waiting")
				grp.tryProduceBlock()
			case <-grp.ProduceRoutineDone:
				group_log.Infof("Produce stop by channel")
				grp.finishProduce()
				return
			}
		}
	}
}

func (grp *ChallengeGroup) AddBlock(block *quorumpb.Block) error {
	group_log.Infof("add block")

	topBlock, err := grp.GetTopBlock()

	if err != nil {
		return err
	}

	valid, err := IsBlockValid(block, topBlock)
	if !valid {
		return err
	}

	if grp.Status == GROUP_DIRTY {
		//is syncing
		group_log.Infof("group dirty, update group db")
		err := grp.applyBlock(block)
		if err != nil {
			return err
		}
	} else {
		if block.ProducerId != grp.ChallengePool[grp.ChallengeIndex[grp.IndexPosition]].Challenger {
			group_log.Infof("Got block from *UNEXPECTED* producer %s", block.ProducerId)
			//reject block
			err = grp.sendNewBlockResp(block, quorumpb.NewBlockRespResult_BLOCK_REJECTED)
			if err != nil {
				return err
			}
			return errors.New("Reject block, received from wrong producer")
		} else {
			group_log.Infof("Got block from producer %s", block.ProducerId)
			topBlock, err := grp.GetTopBlock()
			if err != nil {
				return err
			}

			valid, err := IsBlockValid(block, topBlock)
			if err != nil {
				return err
			}
			//if block is invalid, REJECT block
			if !valid {
				group_log.Infof("Reject block, invalid block")
				err := grp.sendNewBlockResp(block, quorumpb.NewBlockRespResult_BLOCK_REJECTED)
				if err != nil {
					return err
				}
			}

			err = grp.applyBlock(block)
			if err != nil {
				group_log.Infof(err.Error())
				return err
			}

			//block is accepted
			group_log.Info("Accept block")
			err = grp.sendNewBlockResp(block, quorumpb.NewBlockRespResult_BLOCK_ACCEPTED)

			//add myself
			grp.AcceptRecvd++
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (grp *ChallengeGroup) UpdateNewBlockResp(trx *quorumpb.Trx) error {
	group_log.Infof("UpdateNewBlockResp called")

	newBlockResp := &quorumpb.NewBLockResp{}
	if err := proto.Unmarshal(trx.Data, newBlockResp); err != nil {
		return err
	}

	//TODO: should check if receiver is in challenger pool or not
	//TODO: should check if block_id match the block produced
	if newBlockResp.Result == quorumpb.NewBlockRespResult_BLOCK_ACCEPTED {
		grp.AcceptRecvd++
		group_log.Infof("consensus received::%d, needed::%d", grp.AcceptRecvd, grp.IndexLen)
		if grp.AcceptRecvd == grp.IndexLen {
			group_log.Infof("get enough consensus, stop produce")
			grp.stopProduceRoutine()
		}
	}

	return nil
}

func (grp *ChallengeGroup) stopProduceRoutine() {
	grp.ProduceRoutineDone <- true
}

func (grp *ChallengeGroup) applyBlock(block *quorumpb.Block) error {
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

func (grp *ChallengeGroup) produceBlock() {
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

func (grp *ChallengeGroup) finishProduce() {
	group_log.Infof("==================== finish produce ====================")
	//reset status
	grp.initProduce()
}

//ask next block
func (grp *ChallengeGroup) startAskNextBlock() {
	grp.AskNextTicker = time.NewTicker(500 * time.Millisecond)
	grp.AskNextTickerDone = make(chan bool)
	//send ask_next_block every 0.5 sec till get "on_top response"
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

func (grp *ChallengeGroup) stopAskNextBlock() {
	grp.AskNextTicker.Stop()
	grp.AskNextTickerDone <- true
	grp.Status = GROUP_CLEAN
}

func (grp *ChallengeGroup) sendTrxPackage(trx *quorumpb.Trx) error {
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

func (grp *ChallengeGroup) sendNewBlockResp(block *quorumpb.Block, result quorumpb.NewBlockRespResult) error {
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

func (grp *ChallengeGroup) sendBlkPackage(blk *quorumpb.Block) error {
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

func (grp *ChallengeGroup) getChallengeSeed(seed string) (int64, error) {
	num, err := base58.Decode(seed)
	if err != nil {
		return 0, err
	}
	inum := int64(binary.BigEndian.Uint64(num))
	group_log.Infof("seed %d", inum)
	return rand.Int63(), nil
}
