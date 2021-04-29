package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	//"fmt"

	"time"

	"github.com/golang/glog"
	guuid "github.com/google/uuid"
)

type Block struct {
	Cid          string
	GroupId      string
	PrevBlockId  string
	BlockNum     int64
	Timestamp    int64
	Hash         string
	PreviousHash string
	Producer     string
	Signature    string
	Trxs         []Trx
}

func IsBlockValid(newBlock, oldBlock Block) (bool, error) {

	//set hash to ""
	blockWithoutHash := newBlock
	blockWithoutHash.Hash = ""

	/*
		fmt.Printf("%+v\n", blockWithoutHash)

		glog.Infof("Hash 1: %s", CalculateHash(blockWithoutHash))
		glog.Infof("Hash 2：%s", newBlock.Hash)
		glog.Infof("Hash 3: %s", oldBlock.Hash)
	*/

	if CalculateHash(blockWithoutHash) != newBlock.Hash {
		glog.Infof("1")
		return false, nil
	}

	if newBlock.PreviousHash != oldBlock.Hash {
		glog.Infof("2")
		return false, nil
	}

	if newBlock.BlockNum != oldBlock.BlockNum+1 {
		glog.Infof("3")
		return false, nil
	}

	if newBlock.PrevBlockId != oldBlock.Cid {
		glog.Infof("4")
		return false, nil
	}

	return true, nil
}

func CreateBlock(oldBlock Block, trx Trx) Block {
	var newBlock Block
	cid := guuid.New()

	newBlock.Cid = cid.String()
	newBlock.GroupId = oldBlock.GroupId
	newBlock.PrevBlockId = oldBlock.Cid
	newBlock.PreviousHash = oldBlock.Hash
	newBlock.BlockNum = oldBlock.BlockNum + 1
	newBlock.Timestamp = time.Now().UnixNano()
	newBlock.Trxs = append(newBlock.Trxs, trx)
	newBlock.Producer = GetChainCtx().PeerId.Pretty()
	newBlock.Signature = string("Signature from producer")
	newBlock.Hash = ""

	hash := CalculateHash(newBlock)
	newBlock.Hash = hash
	return newBlock
}

func CreateGenesisBlock(groupId string) Block {
	var genesisBlock Block

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
	hash := CalculateHash(genesisBlock)
	genesisBlock.Hash = hash

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
