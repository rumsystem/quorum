package chain

import (
	"errors"
	"math/rand"
	"sort"
	"time"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"google.golang.org/protobuf/proto"

	"github.com/golang/glog"
)

type GroupStatus int8

const (
	GROUP_CLEAN = 0
	GROUP_DIRTY = 1
)

type RoutineStatus int8

const (
	IDLE      = 0
	CHALLENGE = 1
	PRODUCE   = 2
)

type Group struct {
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

	//Produce routine timer
	ChallengeTimer     *time.Timer
	WaitBlockTimer     *time.Timer
	ProduceRoutineDone chan bool

	//Ask next block ticker
	AskNextTicker     *time.Ticker
	AskNextTickerDone chan bool
}

func (grp *Group) init(item *quorumpb.GroupItem) {
	grp.Item = item
	grp.initTrxPool()
	grp.initChallenge()
}

//initial trx pool
func (grp *Group) initTrxPool() {
	grp.TrxPool = make(map[string]*quorumpb.Trx)
}

//initial challenage
func (grp *Group) initChallenge() {
	grp.ChallengePool = make(map[int64]*quorumpb.ChallengeItem)
	grp.ChallengeIndex = nil
	grp.RStatus = IDLE
	grp.IndexPosition = 0
}

//teardown group
func (grp *Group) Teardown() {
	if grp.Status == GROUP_DIRTY {
		grp.stopAskNextBlock()
	}
}

//Start sync group
func (grp *Group) StartSync() error {
	glog.Infof("Group %s start syncing", grp.Item.GroupId)
	grp.Status = GROUP_DIRTY
	grp.startAskNextBlock()
	return nil
}

//Stop sync group
func (grp *Group) StopSync() error {
	glog.Infof("Group stop sync")
	grp.Status = GROUP_CLEAN
	grp.stopAskNextBlock()
	return nil
}

func (grp *Group) GetTopBlock() (*quorumpb.Block, error) {
	return GetDbMgr().GetBlock(grp.Item.LatestBlockId)
}

func (grp *Group) GetBlockId(blockNum int64) (string, error) {
	return GetDbMgr().GetBlkId(blockNum, grp.Item.GroupId)
}

func (grp *Group) CreateGrp(item *quorumpb.GroupItem) error {
	grp.init(item)

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

func (grp *Group) DelGrp() error {
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

func (grp *Group) LeaveGrp() error {
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

/*
//Update group auth
func (grp *Group) UpdAuth() (string, error) {
	glog.Infof("Update auth")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}

	trx, err := CreateTrx(quorumpb.TrxType_AUTH, grp.Item.GroupId, encodedcontent)
	grp.TrxPool[trx.TrxId] = trx
	pbBytes, err := proto.Marshal(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	err = GetChainCtx().GroupTopicPublish(trx.GroupId, pbBytes)
	return trx.TrxId, err
}
*/

/*
	Produce Routine

	POST AND BROADCAST TRX

	If NOT IN PRODUCE ROUTINE
		START A ROUND OF CHALLENGE
	ELSE
		IF RECEIVE CHALLENGE ITEM FROM OTHER NODE
			SET STATUS TO *IN_PRODUCE*
			SEND RESPONSE *ONLY ONCE*

	WAIT 10S FOR INCOMING CHALLENGE RESPONSE
	WHEN TIME UP, SORT AND LOCK CHALLENGE RESPONSE TABLE

	REPEAT TILL PRODUCE DONE OR TIMEOUT OR RUN_OUT_OF CHALLENGE TABLE ITEMS{
		IF I AM LUCKY
			PRODUCE BLOCK
		ELSE
			WAIT 5S INCOMING BLOCK
			IF BLOCK COMES
				IF BLOCK IS VALID
					BREAK
				ELSE
					REJECT AND CONTINUE
			ELSE
				UPDATE CHALLENGE TABLE INDEX
	}

	IF PRODUCE_DONE {
		CHECK INCOMING BLOCK
		IF MY TRX WERE INCLUDED
			UPDATE TRX STATUS AND CLEAN UP
		ELSE
			START NEW PRODUCE ROUTINE AND TRY PACKAGE MY TRX
	} ELSE
		START NEW PRODUCE ROUTINT AND TRY PACKAGE MY TRX
*/

//Add trx to trx pool, prepare for produce block
func (grp *Group) AddTrx(trx *quorumpb.Trx) {
	glog.Infof("Add trx %v", trx)
	grp.TrxPool[trx.TrxId] = trx
}

func (grp *Group) Post(content *quorumpb.Object) (string, error) {
	encodedcontent, err := proto.Marshal(content)
	if err != nil {
		return "", err
	}

	return grp.LanuchProduce(encodedcontent, quorumpb.TrxType_POST)
}

func (grp *Group) UpdAuth(item *quorumpb.BlockListItem) (string, error) {
	glog.Infof("Update Auth")
	encodedcontent, err := proto.Marshal(item)
	if err != nil {
		return "", err
	}
	return grp.LanuchProduce(encodedcontent, quorumpb.TrxType_AUTH)

}

//Post to group (by myself)
func (grp *Group) LanuchProduce(content []byte, trxType quorumpb.TrxType) (string, error) {
	glog.Infof("Launch Produce")
	trx, err := CreateTrx(trxType, grp.Item.GroupId, content)
	err = grp.sendTrxPackage(trx)
	if err != nil {
		return "INVALID_TRX", err
	}

	//add trx to trx pool
	grp.AddTrx(trx)

	//if idle, start a round of challenge
	if grp.RStatus == IDLE {
		var challenge *quorumpb.ChallengeItem
		challenge = &quorumpb.ChallengeItem{}

		challenge.Challenger = GetChainCtx().PeerId.Pretty()
		challenge.ChallengeSeed = rand.Int63n(Base58Decode([]byte(GetChainCtx().PeerId.Pretty())).Int64())

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

		glog.Infof("Start produce routine")
		go grp.startChallenge()
	}

	return trx.TrxId, err
}

//Start a round of challenge
func (grp *Group) startChallenge() {
	glog.Infof("startChallenge")
	grp.RStatus = CHALLENGE
	//set timer for 10s
	grp.ChallengeTimer = time.NewTimer(10 * time.Second)
	grp.ProduceRoutineDone = make(chan bool)
	defer grp.ChallengeTimer.Stop()

	for {
		select {
		case t := <-grp.ChallengeTimer.C:
			//sort challenge list
			glog.Infof("Challenge done, Sort challenge list")
			sort.Slice(grp.ChallengeIndex, func(i, j int) bool {
				return grp.ChallengeIndex[i] < grp.ChallengeIndex[j]
			})
			glog.Infof("challenge pool %v", grp.ChallengeIndex)
			glog.Infof("try produce block " + t.UTC().String())
			go grp.tryProduceBlock()
			return
		case <-grp.ProduceRoutineDone:
			glog.Infof("In challenge, produce routine stopped by channel")
			grp.finishProduce()
		}
	}
}

func (grp *Group) UpdateChallenge(trx *quorumpb.Trx) error {
	glog.Infof("Update challenge")

	challenge := &quorumpb.ChallengeItem{}
	if err := proto.Unmarshal(trx.Data, challenge); err != nil {
		return err
	}

	switch grp.RStatus {
	case IDLE:
		glog.Infof("IDLE, receive challenge item %v", challenge)
		glog.Infof("create and send my challenge response")

		//initial round of challenge
		var myChallenge *quorumpb.ChallengeItem
		myChallenge = &quorumpb.ChallengeItem{}

		myChallenge.Challenger = GetChainCtx().PeerId.Pretty()
		myChallenge.ChallengeSeed = rand.Int63n(Base58Decode([]byte(GetChainCtx().PeerId.Pretty())).Int64())

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
		glog.Infof("CHALLENGE, receive challenge item %v", challenge)
		//add incoming challenge to pool
		grp.ChallengePool[challenge.ChallengeSeed] = challenge
		grp.ChallengeIndex = append(grp.ChallengeIndex, challenge.ChallengeSeed)
	case PRODUCE:
		glog.Infof("in PRODUCE, receive challenge item %v", challenge)
		glog.Infof("ignore challege item")
	}

	return nil
}

func (grp *Group) tryProduceBlock() {
	glog.Infof("try produce block...")

	grp.RStatus = PRODUCE
	index := grp.ChallengeIndex[grp.IndexPosition]

	grp.WaitBlockTimer = time.NewTimer(5 * time.Second)
	defer grp.WaitBlockTimer.Stop()

	//if it is my turn to produce block
	if grp.ChallengePool[index].Challenger == GetChainCtx().PeerId.Pretty() {
		grp.produceBlock()
		glog.Infof("Start wait")
		grp.WaitBlockTimer = time.NewTimer(5 * time.Second)
		for {
			select {
			case t := <-grp.WaitBlockTimer.C:
				glog.Infof("Producer Wait done at " + t.UTC().String())
				//grp.checkResult()
				return
			case <-grp.ProduceRoutineDone:
				glog.Infof("Produce done")
				grp.finishProduce()
				return
			}
		}

	} else {
		glog.Infof("Not my turn, wait block incoming")
		for {
			select {
			case t := <-grp.WaitBlockTimer.C:
				glog.Infof("Wait done at " + t.UTC().String())
				grp.IndexPosition += 1
				glog.Infof("Don't get the block expected, update index %d ", grp.IndexPosition)
				glog.Infof("Start next round of waiting")
				grp.tryProduceBlock()
			case <-grp.ProduceRoutineDone:
				glog.Infof("Produce stop by channel")
				grp.finishProduce()
				return
			}
		}
	}
}

func (grp *Group) AddBlock(block *quorumpb.Block) error {
	glog.Infof("add block")

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
		glog.Infof("group dirty, update group db")
		err := grp.applyBlock(block)
		if err != nil {
			return err
		}
	} else {
		if grp.RStatus == PRODUCE {
			//in producing
			if block.ProducerId != grp.ChallengePool[grp.ChallengeIndex[grp.IndexPosition]].Challenger {
				glog.Infof("Got block from *NOT EXPECTED* producer %s", block.ProducerId)
				return errors.New("Received block from wrong producer")
			} else {
				glog.Infof("Got block from producer %s", block.ProducerId)
				topBlock, err := grp.GetTopBlock()
				if err != nil {
					return err
				}

				valid, err := IsBlockValid(block, topBlock)
				if !valid {
					return err
				}

				err = grp.applyBlock(block)
				if err != nil {
					glog.Infof(err.Error())
					return err
				}

				grp.stopProduceRoutine()
			}
		} else {
			glog.Infof("Not in block produce, ignore incoming block")
		}
	}

	return nil
}

func (grp *Group) stopProduceRoutine() {
	grp.ProduceRoutineDone <- true
}

func (grp *Group) applyBlock(block *quorumpb.Block) error {
	glog.Infof("apply block to group")

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
			glog.Infof("Apply POST trx")
			GetDbMgr().AddPost(trx)
		case quorumpb.TrxType_AUTH:
			glog.Infof("Apply AUTH trx")
			GetDbMgr().UpdateBlkListItem(trx)
		default:
			glog.Infof("Unsupported msgType %s", trx.Type)
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

func (grp *Group) produceBlock() {
	glog.Infof("produce block")

	//get top block
	topBlock, err := grp.GetTopBlock()
	if err != nil {
		glog.Infof(err.Error())
	}

	//package all trx

	glog.Infof("Len %d", len(grp.TrxPool))
	trxs := make([]*quorumpb.Trx, 0, len(grp.TrxPool))
	for _, value := range grp.TrxPool {
		glog.Infof("Append trx")
		trxs = append(trxs, value)
	}

	//create block
	newBlock, err := CreateBlock(topBlock, trxs)
	if err != nil {
		glog.Infof(err.Error())
	}

	//send block via group channel
	grp.sendBlkPackage(newBlock)
	if err != nil {
		glog.Infof(err.Error())
	}
}

/*
TODO:: should count block received number??
func (grp *Group) checkResult() {
	glog.Infof("Check result")
}
*/

func (grp *Group) finishProduce() {
	glog.Infof("finish produce")
	//reset status
	grp.initChallenge()
	grp.initTrxPool()
}

//ask next block
func (grp *Group) startAskNextBlock() {
	grp.AskNextTicker = time.NewTicker(500 * time.Millisecond)
	grp.AskNextTickerDone = make(chan bool)
	//send ask_next_block every 0.5 sec till get "on_top response"
	go func() {
		for {
			select {
			case <-grp.AskNextTickerDone:
				glog.Infof("Ask next block done")
				return
			case t := <-grp.AskNextTicker.C:
				glog.Infof("Ask NEXT_BLOCK " + t.UTC().String())
				//send ask next block msg out
				topBlock, err := grp.GetTopBlock()
				if err != nil {
					glog.Fatalf(err.Error())
				}

				var reqBlockItem quorumpb.ReqBlock
				reqBlockItem.BlockId = topBlock.BlockId
				reqBlockItem.GroupId = grp.Item.GroupId
				reqBlockItem.UserId = GetChainCtx().PeerId.Pretty()

				bItemBytes, err := proto.Marshal(&reqBlockItem)
				if err != nil {
					glog.Warningf(err.Error())
					return
				}

				//send ask next block trx out
				trx, err := CreateTrx(quorumpb.TrxType_REQ_BLOCK, grp.Item.GroupId, bItemBytes)
				if err != nil {
					glog.Warningf(err.Error())
					return
				}

				err = grp.sendTrxPackage(trx)
				if err != nil {
					glog.Warningf(err.Error())
					return
				}
				grp.sendTrxPackage(trx)
			}
		}
	}()
}

func (grp *Group) stopAskNextBlock() {
	grp.AskNextTicker.Stop()
	grp.AskNextTickerDone <- true
	grp.Status = GROUP_CLEAN
}

func (grp *Group) sendTrxPackage(trx *quorumpb.Trx) error {
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

	err = GetChainCtx().GroupTopicPublish(trx.GroupId, pkgBytes)

	if err != nil {
		return err
	}

	return nil
}

func (grp *Group) sendBlkPackage(blk *quorumpb.Block) error {
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
