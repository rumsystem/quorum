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

func CreateBlockByEthKey(parentBlk *quorumpb.Block, consensusInfo *quorumpb.Consensus, trxs []*quorumpb.Trx, groupPublicKey string, keystore localcrypto.Keystore, keyalias string, opts ...string) (*quorumpb.Block, error) {
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

func CreateGenesisBlockByEthKey(groupId string, consensus *quorumpb.Consensus, producerPubkey string) (*quorumpb.Block, error) {
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
	signature, err := ks.EthSignByKeyName(producerPubkey, genesisBlock.BlockHash)
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
		Consensus:      genesisBlock.Consensus,
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
