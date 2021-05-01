package chain

import (
	"crypto/sha256"
	"encoding/hex"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"

	//"fmt"

	"time"

	guuid "github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

func IsBlockValid(newBlock, oldBlock *quorumpb.Block) (bool, error) {

	blockWithoutHash := &quorumpb.Block{}
	//deep copy newBlock by the protobuf. quorumpb.Block is a protobuf defined struct.
	clonedblockbuff, err := proto.Marshal(newBlock)
	if err != nil {
		return false, err
	}
	err = proto.Unmarshal(clonedblockbuff, blockWithoutHash)
	if err != nil {
		return false, err
	}
	//set hash to ""
	blockWithoutHash.Hash = ""

	if CalculateHash(blockWithoutHash) != newBlock.Hash {
		return false, nil
	}

	if newBlock.PreviousHash != oldBlock.Hash {
		return false, nil
	}

	if newBlock.BlockNum != oldBlock.BlockNum+1 {
		return false, nil
	}

	if newBlock.PrevBlockId != oldBlock.Cid {
		return false, nil
	}

	return true, nil
}

func CreateBlock(oldBlock *quorumpb.Block, trx *quorumpb.Trx) *quorumpb.Block {
	var newBlock quorumpb.Block
	cid := guuid.New()

	//deep copy trx by the protobuf. quorumpb.Trx is a protobuf defined struct.
	trxclone := &quorumpb.Trx{}
	clonedtrxbuff, err := proto.Marshal(trx)
	if err != nil {
		return nil
	}
	err = proto.Unmarshal(clonedtrxbuff, trxclone)
	if err != nil {
		return nil
	}

	newBlock.Cid = cid.String()
	newBlock.GroupId = oldBlock.GroupId
	newBlock.PrevBlockId = oldBlock.Cid
	newBlock.PreviousHash = oldBlock.Hash
	newBlock.BlockNum = oldBlock.BlockNum + 1
	newBlock.Timestamp = time.Now().UnixNano()
	newBlock.Trxs = append(newBlock.Trxs, trxclone)
	newBlock.Producer = GetChainCtx().PeerId.Pretty()
	newBlock.Signature = string("Signature from producer")
	newBlock.Hash = ""

	hash := CalculateHash(&newBlock)
	newBlock.Hash = hash
	return &newBlock
}

func CreateGenesisBlock(groupId string) *quorumpb.Block {
	var genesisBlock quorumpb.Block

	cid := guuid.New()
	t := time.Now().UnixNano()

	genesisBlock.Cid = cid.String()
	genesisBlock.GroupId = groupId
	genesisBlock.PrevBlockId = ""
	genesisBlock.PreviousHash = ""
	genesisBlock.BlockNum = 1
	genesisBlock.Timestamp = t
	genesisBlock.Producer = GetChainCtx().PeerId.Pretty()
	genesisBlock.Signature = string("Signature from producer")

	//calculate hash
	hash := CalculateHash(&genesisBlock)
	genesisBlock.Hash = hash

	return &genesisBlock
}

func CalculateHash(block *quorumpb.Block) string {
	bytes, err := proto.Marshal(block)

	if err != nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(bytes))
	hashed := h.Sum(nil)

	return hex.EncodeToString(hashed)
}
