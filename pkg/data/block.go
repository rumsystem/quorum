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

	//"strings"
	"time"
)

func CreateBlockByEthKey(parentBlk *quorumpb.Block, epoch uint64, trxs []*quorumpb.Trx, groupPublicKey string, keystore localcrypto.Keystore, keyalias string, opts ...string) (*quorumpb.Block, error) {
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

// regenerate block with parent info
func RegenrateBlockWithParent(parentBlock *quorumpb.Block, orphanBlock *quorumpb.Block, keystore localcrypto.Keystore, keyalias string, opts ...string) (*quorumpb.Block, error) {
	orphanBlock.PrevHash = parentBlock.BlockHash
	orphanBlock.BlockId = parentBlock.BlockId + 1

	tbytes, err := proto.Marshal(orphanBlock)
	if err != nil {
		return nil, err
	}

	hash := localcrypto.Hash(tbytes)
	orphanBlock.BlockHash = hash

	var signature []byte
	if keyalias == "" {
		signature, err = keystore.EthSignByKeyName(orphanBlock.GroupId, hash, opts...)
	} else {
		signature, err = keystore.EthSignByKeyAlias(keyalias, hash, opts...)
	}

	if err != nil {
		return nil, err
	}

	if len(signature) == 0 {
		return nil, errors.New("create signature failed")
	}

	orphanBlock.ProducerSign = signature
	return orphanBlock, nil
}

func CreateGenesisBlockByEthKey(groupId string, groupPublicKey string, keystore localcrypto.Keystore, keyalias string) (*quorumpb.Block, error) {
	genesisBlock := &quorumpb.Block{
		GroupId:        groupId,
		BlockId:        0,
		PrevHash:       nil,
		ProducerPubkey: groupPublicKey,
		Trxs:           nil,
		TimeStamp:      time.Now().UnixNano(),
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

	//step 1, check hash for newBlock
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

	tbytes, err := proto.Marshal(blkWithOutHashAndSign)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(tbytes)
	if !bytes.Equal(hash, newBlock.BlockHash) {
		return false, fmt.Errorf("hash for new block is invalid")
	}

	//step 2, check blockid and prevhash
	if newBlock.BlockId != parentBlock.BlockId+1 {
		return false, fmt.Errorf("blockid mismatch with parent block")
	}

	if !bytes.Equal(newBlock.PrevHash, parentBlock.BlockHash) {
		return false, errors.New("prevhash mismatch with parent block")
	}

	//step 3, check producer sign
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
	if genesisBlock.BlockId != 0 {
		return false, fmt.Errorf("blockId for genesis block must be 0")
	}

	if genesisBlock.PrevHash != nil {
		return false, fmt.Errorf("prevhash for genesis block must be nil")
	}

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

	bts, err := proto.Marshal(genesisBlockWithoutHashAndSign)
	if err != nil {
		return false, err
	}

	hash := localcrypto.Hash(bts)
	if !bytes.Equal(hash, genesisBlock.BlockHash) {
		return false, fmt.Errorf("hash for new block is invalid")
	}

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
