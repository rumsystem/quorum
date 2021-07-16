package chain

import (
	"bytes"
	"crypto/sha256"
	"errors"

	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

	//"fmt"

	"time"

	guuid "github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func CreateBlock(oldBlock *quorumpb.Block, trxs []*quorumpb.Trx) (*quorumpb.Block, error) {
	var newBlock quorumpb.Block
	blockId := guuid.New()

	//deep copy trx by the protobuf. quorumpb.Trx is a protobuf defined struct.

	newBlock.BlockId = blockId.String()
	newBlock.GroupId = oldBlock.GroupId
	newBlock.PrevBlockId = oldBlock.BlockId
	newBlock.BlockNum = oldBlock.BlockNum + 1
	newBlock.Timestamp = time.Now().UnixNano()
	newBlock.PreviousHash = oldBlock.Hash
	pubkey, err := GetChainCtx().GetPubKey()
	if err != nil {
		return nil, err
	}
	newBlock.ProducerId = pubkey
	newBlock.ProducerPubKey = pubkey

	for _, trx := range trxs {
		trxclone := &quorumpb.Trx{}
		clonedtrxbuff, err := proto.Marshal(trx)

		err = proto.Unmarshal(clonedtrxbuff, trxclone)
		if err != nil {
			return nil, err
		}
		newBlock.Trxs = append(newBlock.Trxs, trxclone)
	}

	hash, err := CalculateHash(&newBlock)
	if err != nil {
		return nil, err
	}
	newBlock.Hash = hash

	signature, err := sign(hash)
	if err != nil {
		return nil, err
	}
	newBlock.Signature = signature

	return &newBlock, nil
}

func CreateGenesisBlock(groupId string) (*quorumpb.Block, error) {
	var genesisBlock quorumpb.Block

	blockId := guuid.New()
	t := time.Now().UnixNano()

	genesisBlock.BlockId = blockId.String()
	genesisBlock.GroupId = groupId
	genesisBlock.PrevBlockId = ""
	genesisBlock.PreviousHash = nil
	genesisBlock.BlockNum = 1
	genesisBlock.Timestamp = t

	pubkey, err := GetChainCtx().GetPubKey()
	if err != nil {
		return nil, err
	}
	genesisBlock.ProducerId = pubkey
	genesisBlock.ProducerPubKey = pubkey
	genesisBlock.Trxs = nil

	//calculate hash
	hash, err := CalculateHash(&genesisBlock)
	if err != nil {
		return nil, err
	}
	genesisBlock.Hash = hash

	signature, err := sign(hash)
	if err != nil {
		return nil, err
	}
	genesisBlock.Signature = signature

	return &genesisBlock, nil
}

func IsBlockValid(newBlock, oldBlock *quorumpb.Block) (bool, error) {
	//deep copy newBlock by the protobuf. quorumpb.Block is a protobuf defined struct.
	clonedblockbuff, err := proto.Marshal(newBlock)
	if err != nil {
		return false, err
	}

	blockWithoutHash := &quorumpb.Block{}
	err = proto.Unmarshal(clonedblockbuff, blockWithoutHash)
	if err != nil {
		return false, err
	}
	//set hash to ""
	blockWithoutHash.Hash = nil
	blockWithoutHash.Signature = nil

	cHash, err := CalculateHash(blockWithoutHash)
	if res := bytes.Compare(cHash, newBlock.Hash); res != 0 {
		return false, errors.New("Hash for new block is invalid")
	}

	if res := bytes.Compare(newBlock.PreviousHash, oldBlock.Hash); res != 0 {
		return false, errors.New("PreviousHash mismatch")
	}

	if newBlock.BlockNum != oldBlock.BlockNum+1 {
		return false, errors.New("BlockNum mismatch")
	}

	if newBlock.PrevBlockId != oldBlock.BlockId {
		return false, errors.New("Previous BlockId mismatch")
	}

	//create pubkey
	serializedpub, err := p2pcrypto.ConfigDecodeKey(newBlock.ProducerPubKey)
	if err != nil {
		return false, err
	}

	pubkey, err := p2pcrypto.UnmarshalPublicKey(serializedpub)
	if err != nil {
		return false, err
	}

	verify, err := pubkey.Verify(newBlock.Hash, newBlock.Signature)
	return verify, err
}

func CalculateHash(block *quorumpb.Block) ([]byte, error) {
	bytes, err := proto.Marshal(block)

	if err != nil {
		return nil, err
	}

	h := sha256.New()
	h.Write([]byte(bytes))
	hashed := h.Sum(nil)

	return hashed, nil
	//return hex.EncodeToString(hashed)
}

func sign(hash []byte) ([]byte, error) {
	signature, err := GetChainCtx().Privatekey.Sign(hash)
	return signature, err
}
