package chain

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/rumsystem/quorum/internal/pkg/conn"
	localcrypto "github.com/rumsystem/quorum/internal/pkg/crypto"
	"github.com/rumsystem/quorum/internal/pkg/logging"
	"github.com/rumsystem/quorum/internal/pkg/nodectx"
	quorumpb "github.com/rumsystem/quorum/internal/pkg/pb"
	"google.golang.org/protobuf/proto"
)

type MolassesUser struct {
	grpItem  *quorumpb.GroupItem
	nodename string
	cIface   ChainMolassesIface
	groupId  string
}

var molauser_log = logging.Logger("user")

func (user *MolassesUser) Init(item *quorumpb.GroupItem, nodename string, iface ChainMolassesIface) {
	molauser_log.Debugf("Init called")
	user.grpItem = item
	user.nodename = nodename
	user.cIface = iface
	user.groupId = item.GroupId
	molauser_log.Infof("<%s> User created", user.groupId)
}

func (user *MolassesUser) sendTrxWithoutRetry(trx *quorumpb.Trx, channel conn.PsConnChanel) (string, error) {
	connMgr, err := conn.GetConn().GetConnMgr(user.groupId)
	if err != nil {
		return "", err
	}

	err = connMgr.SendTrxPubsub(trx, conn.ProducerChannel) /* Q: why hard coded? */
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (user *MolassesUser) sendTrx(trx *quorumpb.Trx, channel conn.PsConnChanel) (string, error) {
	_, err := user.sendTrxWithoutRetry(trx, channel)
	if err != nil {
		return "", err
	}

	err = TrxEnqueue(user.groupId, trx)
	if err != nil {
		return "", err
	}

	return trx.TrxId, nil
}

func (user *MolassesUser) UpdAnnounce(item *quorumpb.AnnounceItem) (string, error) {
	molauser_log.Debugf("<%s> UpdAnnounce called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetAnnounceTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) UpdChainConfig(item *quorumpb.ChainConfigItem) (string, error) {
	molauser_log.Debugf("<%s> UpdChainConfig called", user.groupId)

	trx, err := user.cIface.GetTrxFactory().GetChainConfigTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) UpdSchema(item *quorumpb.SchemaItem) (string, error) {
	molauser_log.Debugf("<%s> UpdSchema called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetUpdSchemaTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) UpdProducer(item *quorumpb.ProducerItem) (string, error) {
	molauser_log.Debugf("<%s> UpdProducer called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetRegProducerTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) UpdUser(item *quorumpb.UserItem) (string, error) {
	molauser_log.Debugf("<%s> UpdUser called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetRegUserTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) UpdAppConfig(item *quorumpb.AppConfigItem) (string, error) {
	molauser_log.Debugf("<%s> UpdAppConfig called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetUpdAppConfigTrx(item)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) PostToGroup(content proto.Message, encryptto ...[]string) (string, error) {
	molauser_log.Debugf("<%s> PostToGroup called", user.groupId)
	trx, err := user.cIface.GetTrxFactory().GetPostAnyTrx(content, encryptto...)
	if err != nil {
		return "", nil
	}
	return user.sendTrx(trx, conn.ProducerChannel)
}

func (user *MolassesUser) AddBlock(block *quorumpb.Block) error {
	molauser_log.Debugf("<%s> AddBlock called", user.groupId)

	//commented by cuicat
	/*
		//check if block is already in chain
		isSaved, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, false, user.nodename)
		if err != nil {
			return err
		}

		if isSaved {
			return fmt.Errorf("Block %s already saved, ignore", block.BlockId)
		}
	*/

	//check if block is in cache
	isCached, err := nodectx.GetDbMgr().IsBlockExist(block.BlockId, true, user.nodename)
	if err != nil {
		return err
	}

	if isCached {
		molauser_log.Debugf("<%s> cached block, update block", user.groupId)
	}

	//Save block to cache
	err = nodectx.GetDbMgr().AddBlock(block, true, user.nodename)
	if err != nil {
		return err
	}

	//check if parent of block exist
	parentExist, err := nodectx.GetDbMgr().IsParentExist(block.PrevBlockId, false, user.nodename)
	if err != nil {
		return err
	}

	if !parentExist {
		molauser_log.Debugf("<%s> parent of block <%s> is not exist", user.groupId, block.BlockId)
		return errors.New("PARENT_NOT_EXIST")
	}

	//get parent block
	parentBlock, err := nodectx.GetDbMgr().GetBlock(block.PrevBlockId, false, user.nodename)
	if err != nil {
		return err
	}

	//valid block with parent block
	valid, err := IsBlockValid(block, parentBlock)
	if !valid {
		molauser_log.Debugf("<%s> remove invalid block <%s> from cache", user.groupId, block.BlockId)
		molauser_log.Warningf("<%s> invalid block <%s>", user.groupId, err.Error())
		return nodectx.GetDbMgr().RmBlock(block.BlockId, true, user.nodename)
	}

	//search cache, gather all blocks can be connected with this block
	blocks, err := nodectx.GetDbMgr().GatherBlocksFromCache(block, true, user.nodename)
	if err != nil {
		return err
	}

	//get all trxs from blocks
	var trxs []*quorumpb.Trx
	trxs, err = GetAllTrxs(blocks)
	if err != nil {
		return err
	}

	//apply trxs
	err = user.applyTrxs(trxs, user.nodename)
	if err != nil {
		return err
	}

	//move gathered blocks from cache to chain
	for _, block := range blocks {
		molauser_log.Debugf("<%s> move block <%s> from cache to chain", user.groupId, block.BlockId)
		err := nodectx.GetDbMgr().AddBlock(block, false, user.nodename)
		if err != nil {
			return err
		}

		err = nodectx.GetDbMgr().RmBlock(block.BlockId, true, user.nodename)
		if err != nil {
			return err
		}
	}

	//update block produced count
	for _, block := range blocks {
		err := nodectx.GetDbMgr().AddProducedBlockCount(user.groupId, block.ProducerPubKey, user.nodename)
		if err != nil {
			return err
		}
	}

	//calculate new height
	molauser_log.Debugf("<%s> height before recal <%d>", user.groupId, user.grpItem.HighestHeight)
	topBlock, err := nodectx.GetDbMgr().GetBlock(user.grpItem.HighestBlockId, false, user.nodename)
	if err != nil {
		return err
	}
	newHeight, newHighestBlockId, err := RecalChainHeight(blocks, user.grpItem.HighestHeight, topBlock, user.nodename)
	if err != nil {
		return err
	}
	molauser_log.Debugf("<%s> new height <%d>, new highest blockId %v", user.groupId, newHeight, newHighestBlockId)

	//if the new block is not highest block after recalculate, we need to "trim" the chain
	if newHeight < user.grpItem.HighestHeight {

		//from parent of the new blocks, get all blocks not belong to the longest path
		resendBlocks, err := GetTrimedBlocks(blocks, user.nodename)
		if err != nil {
			return err
		}

		var resendTrxs []*quorumpb.Trx
		resendTrxs, err = GetMyTrxs(resendBlocks, user.nodename, user.grpItem.UserSignPubkey)

		if err != nil {
			return err
		}

		UpdateResendCount(resendTrxs)
		err = user.resendTrx(resendTrxs)
	}

	return user.cIface.UpdChainInfo(newHeight, newHighestBlockId)
}

//resend all trx in the list
func (user *MolassesUser) resendTrx(trxs []*quorumpb.Trx) error {
	molauser_log.Debugf("<%s> resendTrx called", user.groupId)
	for _, trx := range trxs {
		molauser_log.Debugf("<%s> resend Trx <%s>", user.groupId, trx.TrxId)
		user.sendTrx(trx, conn.ProducerChannel)
	}
	return nil
}

func (user *MolassesUser) applyTrxs(trxs []*quorumpb.Trx, nodename string) error {
	molauser_log.Debugf("<%s> applyTrxs called", user.groupId)
	for _, trx := range trxs {
		//check if trx already applied
		isExist, err := nodectx.GetDbMgr().IsTrxExist(trx.TrxId, trx.Nonce, nodename)
		if err != nil {
			molauser_log.Debugf("<%s> %s", user.groupId, err.Error())
			continue
		}

		if isExist {
			molauser_log.Debugf("<%s> trx <%s> existed, update trx only", user.groupId, trx.TrxId)
			nodectx.GetDbMgr().AddTrx(trx)
			continue
		}

		originalData := trx.Data

		//new trx, apply it
		if trx.Type == quorumpb.TrxType_POST && user.grpItem.EncryptType == quorumpb.GroupEncryptType_PRIVATE {
			//for post, private group, encrypted by pgp for all announced group user
			ks := localcrypto.GetKeystore()
			decryptData, err := ks.Decrypt(user.grpItem.GroupId, trx.Data)
			if err != nil {
				trx.Data = []byte("")
				//return err
			} else {
				//set trx.Data to decrypted []byte
				trx.Data = decryptData
			}

		} else {
			//decode trx data
			ciperKey, err := hex.DecodeString(user.grpItem.CipherKey)
			if err != nil {
				return err
			}

			decryptData, err := localcrypto.AesDecode(trx.Data, ciperKey)
			if err != nil {
				return err
			}

			//set trx.Data to decrypted []byte
			trx.Data = decryptData
		}

		//apply trx
		molauser_log.Debugf("<%s> try apply trx <%s>", user.groupId, trx.TrxId)

		//check if snapshotTag is available
		if trx.Type != quorumpb.TrxType_POST {
			snapshotTag, err := nodectx.GetDbMgr().GetSnapshotTag(trx.GroupId, nodename)
			if err == nil && snapshotTag != nil {
				if snapshotTag.HighestHeight > user.grpItem.HighestHeight {
					molauser_log.Debugf("<%s> snapshotTag exist, trx already applied, ignore <%s>", user.groupId, trx.TrxId)
					continue
				}
			}
		}

		switch trx.Type {
		case quorumpb.TrxType_POST:
			molauser_log.Debugf("<%s> apply POST trx", user.groupId)
			nodectx.GetDbMgr().AddPost(trx, nodename)
		case quorumpb.TrxType_PRODUCER:
			molauser_log.Debugf("<%s> apply PRODUCER trx", user.groupId)
			nodectx.GetDbMgr().UpdateProducerTrx(trx, nodename)
			user.cIface.UpdProducerList()
			user.cIface.CreateConsensus()
		case quorumpb.TrxType_USER:
			molauser_log.Debugf("<%s> apply USER trx", user.groupId)
			nodectx.GetDbMgr().UpdateUserTrx(trx, nodename)
			user.cIface.UpdUserList()
		case quorumpb.TrxType_ANNOUNCE:
			molauser_log.Debugf("<%s> apply ANNOUNCE trx", user.groupId)
			nodectx.GetDbMgr().UpdateAnnounceTrx(trx, nodename)
		case quorumpb.TrxType_SCHEMA:
			molauser_log.Debugf("<%s> apply SCHEMA trx", user.groupId)
			nodectx.GetDbMgr().UpdateSchema(trx, nodename)
		case quorumpb.TrxType_ASK_PEERID_RESP:
			molauser_log.Debugf("<%s> handle ASK_PEERID_RESP trx", user.groupId)
		case quorumpb.TrxType_APP_CONFIG:
			molauser_log.Debugf("<%s> apply APP_CONFIG trx", user.groupId)
			nodectx.GetDbMgr().UpdateAppConfigTrx(trx, nodename)
		case quorumpb.TrxType_CHAIN_CONFIG:
			molauser_log.Debugf("<%s> apply CHAIN_CONFIG trx", user.groupId)
			err := nodectx.GetDbMgr().UpdateChainConfigTrx(trx, nodename)
			if err != nil {
				fmt.Println(err.Error())
			}
		default:
			molauser_log.Warningf("<%s> unsupported msgType <%s>", user.groupId, trx.Type)
		}

		//set trx data to original(encrypted)
		trx.Data = originalData

		//save trx to db
		nodectx.GetDbMgr().AddTrx(trx, nodename)
	}
	return nil
}
