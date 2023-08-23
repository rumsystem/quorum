package data

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	localcrypto "github.com/rumsystem/quorum/pkg/crypto"
	quorumpb "github.com/rumsystem/quorum/pkg/pb"
	"google.golang.org/protobuf/proto"

	"time"
)

func CreateBlockByEthKey(parentBlk *quorumpb.Block, consensusInfo *quorumpb.Consensus, trxs []*quorumpb.Trx, producerPubkey, producerKeyname string, opts ...string) (*quorumpb.Block, error) {
	newBlock := &quorumpb.Block{
		GroupId:        parentBlk.GroupId,
		BlockId:        parentBlk.BlockId + 1,
		PrevHash:       parentBlk.BlockHash,
		ProducerPubkey: producerPubkey,
		Trxs:           trxs,
		TimeStamp:      time.Now().UnixNano(),
	}

	tbytes, err := proto.Marshal(newBlock)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(tbytes)
	newBlock.BlockHash = hash

	ks := localcrypto.GetKeystore()
	signature, err := ks.EthSignByKeyName(producerKeyname, hash, opts...)
	if err != nil {
		return nil, err
	}

	newBlock.ProducerSign = signature
	return newBlock, nil
}

func CreateGenesisBlockByEthKey(groupId string, consensus *quorumpb.Consensus, producerPubkey, producerKeyName string) (*quorumpb.Block, error) {
	genesisBlock := &quorumpb.Block{
		BlockId:        0,
		GroupId:        groupId,
		PrevHash:       nil,
		Trxs:           []*quorumpb.Trx{},
		TimeStamp:      time.Now().UnixNano(),
		ProducerPubkey: producerPubkey,
		Consensus:      consensus,
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	bbytes, err := proto.Marshal(genesisBlock)
	if err != nil {
		return nil, err
	}

	genesisBlock.BlockHash = localcrypto.Hash(bbytes)

	ks := localcrypto.GetKeystore()
	signature, err := ks.EthSignByKeyName(producerKeyName, genesisBlock.BlockHash)
	if err != nil {
		return nil, err
	}

	genesisBlock.ProducerSign = signature
	return genesisBlock, nil
}

func ValidBlockWithParent(newBlock, parentBlock *quorumpb.Block) (bool, error) {
	//dump block without hash and sign
	blkWithOutHashAndSign := &quorumpb.Block{
		GroupId:        newBlock.GroupId,
		BlockId:        newBlock.BlockId,
		PrevHash:       newBlock.PrevHash,
		ProducerPubkey: newBlock.ProducerPubkey,
		Trxs:           newBlock.Trxs,
		TimeStamp:      newBlock.TimeStamp,
		Consensus:      newBlock.Consensus,
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	//get hash
	tbytes, err := proto.Marshal(blkWithOutHashAndSign)
	if err != nil {
		return false, err
	}
	hash := localcrypto.Hash(tbytes)

	//check hash for block
	if !bytes.Equal(hash, newBlock.BlockHash) {
		return false, fmt.Errorf("hash for new block is invalid")
	}

	//check blockid
	if newBlock.BlockId != parentBlock.BlockId+1 {
		return false, fmt.Errorf("blockid mismatch with parent block")
	}

	//check prevhash
	if !bytes.Equal(newBlock.PrevHash, parentBlock.BlockHash) {
		return false, errors.New("prevhash mismatch with parent block")
	}

	//verify producer sign
	bytespubkey, err := base64.RawURLEncoding.DecodeString(newBlock.ProducerPubkey)
	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			ks := localcrypto.GetKeystore()
			r := ks.EthVerifySign(hash, newBlock.ProducerSign, ethpubkey)
			return r, nil
		}
		return false, err
	}

	return true, nil
}

func ValidGenesisBlockPoa(genesisBlock *quorumpb.Block) (bool, error) {
	//check blockid is 0
	if genesisBlock.BlockId != 0 {
		return false, fmt.Errorf("blockId for genesis block must be 0")
	}

	//check prevhash is nil
	if genesisBlock.PrevHash != nil {
		return false, fmt.Errorf("prevhash for genesis block must be nil")
	}

	if genesisBlock.Consensus == nil {
		return false, fmt.Errorf("consensus info for genesis block must not be nil")
	}

	if genesisBlock.Consensus.Type != quorumpb.GroupConsenseType_POA {
		return false, fmt.Errorf("consensus type for genesis block must be poa")
	}

	//convert to POAConsensus
	poaConsensus := &quorumpb.PoaConsensusInfo{}
	err := proto.Unmarshal(genesisBlock.Consensus.Data, poaConsensus)
	if err != nil {
		return false, err
	}

	if poaConsensus.ConsensusId == "" ||
		poaConsensus.ChainVer != 0 ||
		poaConsensus.ForkInfo == nil ||
		poaConsensus.InTrx != "" {
		return false, fmt.Errorf("consensus info for genesis block is invalid")
	}

	forkInfo := poaConsensus.ForkInfo

	//check consensus info
	if forkInfo.GroupId != genesisBlock.GroupId ||
		forkInfo.EpochDuration <= 500 ||
		forkInfo.StartFromBlock != 0 ||
		forkInfo.StartFromEpoch != 0 ||
		forkInfo.Producers == nil ||
		len(forkInfo.Producers) != 1 {
		return false, fmt.Errorf("consensus info for genesis block is invalid")
	}

	blockClone := proto.Clone(genesisBlock).(*quorumpb.Block)
	blockClone.BlockHash = nil
	blockClone.ProducerSign = nil

	//get hash
	bts, err := proto.Marshal(blockClone)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(bts)

	//check hash for block
	if !bytes.Equal(hash, genesisBlock.BlockHash) {
		return false, fmt.Errorf("hash for new block is invalid")
	}

	return VerifySign(genesisBlock.ProducerPubkey, genesisBlock.BlockHash, genesisBlock.ProducerSign)
}
