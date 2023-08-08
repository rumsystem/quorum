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

func CreateBlockByEthKey(parentBlk *quorumpb.Block, trxs []*quorumpb.Trx, groupPublicKey string, keystore localcrypto.Keystore, keyalias string, opts ...string) (*quorumpb.Block, error) {
	newBlock := &quorumpb.Block{
		GroupId:        parentBlk.GroupId,
		BlockId:        parentBlk.BlockId + 1,
		PrevHash:       parentBlk.BlockHash,
		ProducerPubkey: groupPublicKey,
		Trxs:           trxs,
		TimeStamp:      time.Now().UnixNano(),
	}

	tbytes, err := proto.Marshal(newBlock)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(tbytes)
	newBlock.BlockHash = hash

	var signature []byte
	if keyalias == "" {
		signature, err = keystore.EthSignByKeyName(newBlock.GroupId, hash, opts...)
	} else {
		signature, err = keystore.EthSignByKeyAlias(keyalias, hash, opts...)
	}

	if err != nil {
		return nil, err
	}

	if len(signature) == 0 {
		return nil, errors.New("create signature failed")
	}

	newBlock.ProducerSign = signature
	return newBlock, nil
}

func CreateGenesisBlockByEthKey(groupId string, producerPubkey string, keystore localcrypto.Keystore, keyalias string) (*quorumpb.Block, error) {
	genesisBlock := &quorumpb.Block{
		GroupId:        groupId,
		BlockId:        0,
		PrevHash:       nil,
		ProducerPubkey: producerPubkey,
		Trxs:           nil,
		TimeStamp:      time.Now().UnixNano(),
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	bbytes, err := proto.Marshal(genesisBlock)
	if err != nil {
		return nil, err
	}

	blockHash := localcrypto.Hash(bbytes)
	genesisBlock.BlockHash = blockHash

	var signature []byte
	if keyalias == "" {
		signature, err = keystore.EthSignByKeyName(genesisBlock.GroupId, blockHash)
	} else {
		signature, err = keystore.EthSignByKeyAlias(keyalias, blockHash)
	}
	if err != nil {
		return nil, err
	}
	if len(signature) == 0 {
		return nil, errors.New("create signature on genesisblock failed")
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

func ValidGenesisBlock(genesisBlock *quorumpb.Block) (bool, error) {
	//check blockid is 0
	if genesisBlock.BlockId != 0 {
		return false, fmt.Errorf("blockId for genesis block must be 0")
	}

	//check prevhash is nil
	if genesisBlock.PrevHash != nil {
		return false, fmt.Errorf("prevhash for genesis block must be nil")
	}

	//dump block without hash and sign
	genesisBlockWithoutHashAndSign := &quorumpb.Block{
		GroupId:        genesisBlock.GroupId,
		BlockId:        genesisBlock.BlockId,
		PrevHash:       genesisBlock.PrevHash,
		ProducerPubkey: genesisBlock.ProducerPubkey,
		Trxs:           genesisBlock.Trxs,
		TimeStamp:      genesisBlock.TimeStamp,
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	//get hash
	bts, err := proto.Marshal(genesisBlockWithoutHashAndSign)
	if err != nil {
		return false, err
	}
	hash := localcrypto.Hash(bts)

	//check hash for block
	if !bytes.Equal(hash, genesisBlock.BlockHash) {
		return false, fmt.Errorf("hash for new block is invalid")
	}

	//verify producer sign
	bytespubkey, err := base64.RawURLEncoding.DecodeString(genesisBlock.ProducerPubkey)
	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			ks := localcrypto.GetKeystore()
			r := ks.EthVerifySign(hash, genesisBlock.ProducerSign, ethpubkey)
			return r, nil
		}
		return false, err
	}

	return true, nil
}

func CreateGenesisBlockRumLiteByEthKey(groupId string, ownerPubkey, ownerKeyName string, consensus *quorumpb.ConsensusInfoRumLite) (*quorumpb.BlockRumLite, error) {
	ks := localcrypto.GetKeystore()
	genesisBlockRumLite := &quorumpb.BlockRumLite{
		BlockId:        0,
		GroupId:        groupId,
		PrevHash:       nil,
		Trxs:           nil,
		TimeStamp:      time.Now().UnixNano(),
		ProducerPubkey: ownerPubkey,
		ConsensusInfo:  consensus,
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	bbytes, err := proto.Marshal(genesisBlockRumLite)
	if err != nil {
		return nil, err
	}

	genesisBlockRumLite.BlockHash = localcrypto.Hash(bbytes)

	signature, err := ks.EthSignByKeyName(ownerKeyName, genesisBlockRumLite.BlockHash)
	if err != nil {
		return nil, err
	}

	genesisBlockRumLite.ProducerSign = signature
	return genesisBlockRumLite, nil
}

func ValidGenesisBlockRumLite(genesisBlock *quorumpb.BlockRumLite) (bool, error) {
	//check blockid is 0
	if genesisBlock.BlockId != 0 {
		return false, fmt.Errorf("blockId for genesis block must be 0")
	}

	//check prevhash is nil
	if genesisBlock.PrevHash != nil {
		return false, fmt.Errorf("prevhash for genesis block must be nil")
	}

	if genesisBlock.ConsensusInfo == nil {
		return false, fmt.Errorf("consensus info for genesis block must not be nil")
	}

	if genesisBlock.ConsensusInfo.Poa == nil {
		return false, fmt.Errorf("consensus type for genesis block must be poa")
	}

	//check consensus info
	if genesisBlock.ConsensusInfo.Poa.EpochDuration <= 0 ||
		genesisBlock.ConsensusInfo.Poa.CurrBlockId != 0 ||
		genesisBlock.ConsensusInfo.Poa.ChainVer != 0 ||
		genesisBlock.ConsensusInfo.Poa.CurrEpoch != 0 ||
		genesisBlock.ConsensusInfo.Poa.ConsensusId == "" ||
		genesisBlock.ConsensusInfo.Poa.Producers == nil ||
		len(genesisBlock.ConsensusInfo.Poa.Producers) != 1 {
		return false, fmt.Errorf("consensus info for genesis block is invalid")
	}

	//dump block without hash and sign
	genesisBlockWithoutHashAndSign := &quorumpb.BlockRumLite{
		BlockId:        genesisBlock.BlockId,
		GroupId:        genesisBlock.GroupId,
		PrevHash:       genesisBlock.PrevHash,
		Trxs:           genesisBlock.Trxs,
		TimeStamp:      genesisBlock.TimeStamp,
		ProducerPubkey: genesisBlock.ProducerPubkey,
		ConsensusInfo:  genesisBlock.ConsensusInfo,
		BlockHash:      nil,
		ProducerSign:   nil,
	}

	//get hash
	bts, err := proto.Marshal(genesisBlockWithoutHashAndSign)
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

func VerifySign(key string, hash, sign []byte) (bool, error) {
	//verify signature
	ks := localcrypto.GetKeystore()
	bytespubkey, err := base64.RawURLEncoding.DecodeString(key)
	if err == nil { //try eth key
		ethpubkey, err := ethcrypto.DecompressPubkey(bytespubkey)
		if err == nil {
			r := ks.EthVerifySign(hash, sign, ethpubkey)
			return r, nil
		}
		return false, err
	}
	return false, err
}
