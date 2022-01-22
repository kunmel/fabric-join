package kvledger

import "github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"

// NEW File
// everything is new

type RollbackInterface interface {
	CrossRollbackOrigVal(kov *[]statedb.KeyOrigVal)
}

type CrossGetStateInterface interface {
	// GetStateNoRSet(namespace string, key string) (*statedb.VersionedValue, error)
	GetStateNoRSet(namespace string, key string) ([]byte, error)
}
