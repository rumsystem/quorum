package appdata

import (
	//"fmt"
	badger "github.com/dgraph-io/badger/v3"
	chain "github.com/huo-ju/quorum/internal/pkg/chain"
	quorumpb "github.com/huo-ju/quorum/internal/pkg/pb"
	logging "github.com/ipfs/go-log/v2"
)

var appdatalog = logging.Logger("appdata")

type AppDb struct {
	Db       *badger.DB
	DataPath string
}

func InitDb(datapath string, dbopts *chain.DbOption) *AppDb {
	newdb, err := badger.Open(badger.DefaultOptions(datapath + "_appdb").WithValueLogFileSize(dbopts.LogFileSize).WithMemTableSize(dbopts.MemTableSize).WithValueLogMaxEntries(dbopts.LogMaxEntries).WithBlockCacheSize(dbopts.BlockCacheSize).WithCompression(dbopts.Compression))
	if err != nil {
		appdatalog.Fatal(err.Error())
	}
	appdatalog.Infof("Appdb initialized")
	return &AppDb{Db: newdb, DataPath: datapath}
}

func (appdb *AppDb) AddMetaByTrx(trx *quorumpb.Trx) error {

	//Index 1: order by insert sequence
	//
	return nil
}
