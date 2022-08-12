package consensus

import (

	//p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	"github.com/rumsystem/quorum/pkg/consensus/def"
	quorumpb "github.com/rumsystem/rumchaindata/pkg/pb"
)

var molaproducer_log = logging.Logger("producer")

type MolassesProducer struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   def.ChainMolassesIface
	groupId  string
	bft      *Bft
}

func (producer *MolassesProducer) Init(item *quorumpb.GroupItem, nodename string, iface def.ChainMolassesIface) {
	molaproducer_log.Debug("Init called")
	producer.grpItem = item
	producer.cIface = iface
	producer.nodename = nodename
	producer.groupId = item.GroupId

	config, err := producer.createBftConfig()
	if err != nil {
		molaproducer_log.Errorf("create bft failed")
		molauser_log.Error(err.Error())
		return
	}

	producer.bft = NewBft(*config, producer.groupId)
	molaproducer_log.Infof("<%s> producer created", producer.groupId)
}

func (producer *MolassesProducer) createBftConfig() (*Config, error) {
	producer_nodes, err := nodectx.GetNodeCtx().GetChainStorage().GetProducers(producer.groupId, producer.nodename)
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, producer := range producer_nodes {
		nodes = append(nodes, producer.ProducerPubkey)
	}

	n := len(nodes)
	f := (n - 1) / 3

	scalar := 20
	batchSize := (len(nodes) * 2) * scalar

	config := &Config{
		N:            n,
		F:            f,
		Nodes:        nodes,
		BatchSize:    batchSize,
		MyNodePubkey: producer.grpItem.UserSignPubkey,
		MySignPubkey: producer.grpItem.UserSignPubkey,
	}

	return config, nil
}

// Add trx to trx pool
func (producer *MolassesProducer) AddTrx(trx *quorumpb.Trx) {
	molaproducer_log.Debugf("<%s> AddTrx called", producer.groupId)

	//check if trx sender is in group block list
	isAllow, err := nodectx.GetNodeCtx().GetChainStorage().CheckTrxTypeAuth(trx.GroupId, trx.SenderPubkey, trx.Type, producer.nodename)
	if err != nil {
		return
	}

	if !isAllow {
		molaproducer_log.Debugf("<%s> user <%s> don't has permission on trx type <%s>", producer.groupId, trx.SenderPubkey, trx.Type.String())
		return
	}

	//check if trx with same nonce exist, !!Only applied to client which support nonce
	isExist, err := nodectx.GetNodeCtx().GetChainStorage().IsTrxExist(trx.TrxId, trx.Nonce, producer.nodename)
	if isExist {
		molaproducer_log.Debugf("<%s> Trx <%s> with nonce <%d> already packaged, ignore <%s>", producer.groupId, trx.TrxId, trx.Nonce)
		return
	}

	molaproducer_log.Debugf("<%s> Molasses AddTrx called, add trx <%s>", producer.groupId, trx.TrxId)
	err = producer.bft.AddTrx(trx)
	if err != nil {
		molaproducer_log.Errorf("add trx failed %s", err.Error())
	}
}

func (producer *MolassesProducer) HandleHBMsg(hbmsg *quorumpb.HBMsg) error {
	return producer.bft.HandleMessage(hbmsg)
}

/*

func (producer *MolassesProducer) startMergeBlock() error {
	molaproducer_log.Debugf("<%s> startMergeBlock called", producer.groupId)

	defer func() {
		molaproducer_log.Infof("<%s> set StatusIdle", producer.groupId)
		producer.status = StatusIdle
		producer.statusmu.Unlock()

		//since sync.map don't have len(), count manually
		var count uint
		producer.trxPool.Range(func(key interface{}, value interface{}) bool {
			count++
			return true
		})

		if count != 0 {
			molaproducer_log.Debugf("<%s> start produce block", producer.groupId)
			producer.startProduceBlock()
		}
	}()
	molaproducer_log.Debugf("<%s> set merge timer to <%d>s", producer.groupId, MERGE_TIMER)
	mergeTimer := time.NewTimer(MERGE_TIMER * time.Second)
	t := <-mergeTimer.C
	molaproducer_log.Debugf("<%s> merge timer ticker...<%s>", producer.groupId, t.UTC().String())

	candidateBlkid := ""
	var oHash []byte
	for _, blk := range producer.blockPool {
		nHash := sha256.Sum256(blk.Signature)
		//comparing two hash bytes lexicographically
		if bytes.Compare(oHash[:], nHash[:]) == -1 { //-1 means ohash < nhash, and we want keep the larger one
			candidateBlkid = blk.BlockId
			oHash = nHash[:]
		}
	}

	molaproducer_log.Debugf("<%s> candidate block decided, block Id : %s", producer.groupId, candidateBlkid)

	surfix := ""
	if producer.blockPool[candidateBlkid].ProducerPubKey == producer.grpItem.OwnerPubKey {
		surfix = "OWNER"
	} else {
		surfix = "PRODUCER"
	}

	molaproducer_log.Debugf("<%s> winner <%s> (%s)", producer.groupId, producer.blockPool[candidateBlkid].ProducerPubKey, surfix)
	err := producer.cIface.AddBlock(producer.blockPool[candidateBlkid])

	if err != nil {
		molaproducer_log.Errorf("<%s> save block <%s> error <%s>", producer.groupId, candidateBlkid, err)
		if err.Error() == "PARENT_NOT_EXIST" {
			molaproducer_log.Debugf("<%s> parent not found, sync backward for missing blocks from <%s>", producer.groupId, candidateBlkid, err)
			return producer.cIface.GetChainSyncIface().SyncBackward(candidateBlkid, producer.nodename)
		}
	} else {
		molaproducer_log.Debugf("<%s> block saved", producer.groupId)
		//check if I am the winner
		if producer.blockPool[candidateBlkid].ProducerPubKey == producer.grpItem.UserSignPubkey {
			molaproducer_log.Debugf("<%s> winner send new block out", producer.groupId)

			connMgr, err := conn.GetConn().GetConnMgr(producer.groupId)
			if err != nil {
				return err
			}
			err = connMgr.SendBlockPsconn(producer.blockPool[candidateBlkid], conn.UserChannel)
			if err != nil {
				molaproducer_log.Warnf("<%s> <%s>", producer.groupId, err.Error())
			}
		}
	}

	molaproducer_log.Debugf("<%s> merge done", producer.groupId)
	producer.blockPool = make(map[string]*quorumpb.Block)

	return nil
}

*/
