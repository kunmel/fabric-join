package crossleveldb

import (
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/common/ledger/util/leveldbhelper"
	"github.com/hyperledger/fabric/core/ledger/kvledger/crossdb"
	"github.com/hyperledger/fabric/core/ledger/ledgerconfig"
)

type crossdbLogger interface {
	Debugf(template string, args ...interface{})
	Errorf(template string, args ...interface{})
	Infof(template string, args ...interface{})
	Warningf(template string, args ...interface{})
}

var logger crossdbLogger = flogging.MustGetLogger("crossleveldb")

type CrossDBProvider struct {
	dbProvider *leveldbhelper.Provider
}


func NewCrossDBProvider() *CrossDBProvider{
	dbPath := ledgerconfig.GetCrossLevelDBPath()
	logger.Infof("constructing CatDBProvider dbPath = %s", dbPath)
	dbProvider := leveldbhelper.NewProvider(&leveldbhelper.Conf{DBPath: dbPath})
	return &CrossDBProvider{dbProvider}
}

func (provider *CrossDBProvider) GetDBHandler(dbName string) (crossdb.CrossDB, error){
	return newCrossDB(provider.dbProvider.GetDBHandle(dbName), dbName), nil
}

func (provider *CrossDBProvider) Close() {
	provider.dbProvider.Close()
}

type crossDB struct{
	db *leveldbhelper.DBHandle
	dbName string
}

func newCrossDB(db *leveldbhelper.DBHandle, dbName string) *crossDB{
	return &crossDB{db, dbName}
}

func (crossDB *crossDB) Get(key []byte) ([]byte, error) {
	return crossDB.db.Get(key)
}

func (crossDB *crossDB) Put(key []byte, value []byte, sync bool) error {
	return crossDB.db.Put(key, value, sync)
}


func (crossDB *crossDB) Test(){
	logger.Infof("[CrossDB]Testing")
}
