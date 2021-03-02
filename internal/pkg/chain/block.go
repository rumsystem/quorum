package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	guuid "github.com/google/uuid"
)

type Block struct {
	Cid          string
	GroupId      string
	prevBlockId  string
	hash         string
	previousHash string
	blockNum     int64
	timestamp    int64
	trxs         []Trx
}

func IsBlockValid(newBlock, oldBlock Block) (bool, error) {
	blockWithoutHash := newBlock

	//set hash to ""
	blockWithoutHash.hash = ""

	if CalculateHash(blockWithoutHash) != newBlock.hash {
		return false, nil
	}

	if newBlock.previousHash != oldBlock.hash {
		return false, nil
	}

	if newBlock.blockNum != oldBlock.blockNum+1 {
		return false, nil
	}

	if newBlock.prevBlockId != oldBlock.Cid {
		return false, nil
	}

	//verify all trx signatures

	return true, nil
}

func CreateBlock(oldBlock Block, trx Trx) Block {
	var newBlock Block
	cid := guuid.New()
	t := time.Now().UnixNano()

	newBlock.Cid = cid.String()
	newBlock.GroupId = TestGroupId
	newBlock.prevBlockId = oldBlock.Cid
	newBlock.previousHash = oldBlock.hash
	newBlock.blockNum = oldBlock.blockNum + 1
	newBlock.timestamp = t

	hash := CalculateHash(newBlock)
	newBlock.hash = hash
	return newBlock
}

func CreateGenesisBlock() Block {
	var genesisBlock Block

	cid := guuid.New()
	t := time.Now().UnixNano()

	genesisBlock.Cid = cid.String()
	genesisBlock.GroupId = TestGroupId
	genesisBlock.prevBlockId = ""
	genesisBlock.previousHash = ""
	genesisBlock.blockNum = 1
	genesisBlock.timestamp = t

	//calculate hash
	hash := CalculateHash(genesisBlock)
	genesisBlock.hash = hash

	return genesisBlock
}

func CalculateHash(block Block) string {
	bytes, err := json.Marshal(&block)

	if err != nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(bytes))
	hashed := h.Sum(nil)

	return hex.EncodeToString(hashed)
}
