package cross

import (
	"fmt"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

// 多链协同跨链，协同跨链结果为成功或失败
// 由orderer发送给peers协同跨链结果
// 根据协同跨链结果，peers处理跨链相关内存变量，及跨链数据库数据
// 成功-只更新数据库&删除内存相应跨链信息；失败-回滚statedb&更新数据库&删除跨链相关信息

/*
//type CrossH crossinterface.CrossHandler
type CrossH struct {}

func (h *CrossH)CrossConfirmSucc(txid string) error {
	processCrossVar(txid)
	//跨链数据库更新成功
	processCrossDB(txid, "Mult-Succ")

	return nil
}
//func (h *crossinterface.CrossHandler)CrossConfirmFail(channelid string, txid string) error{
func (h CrossH)CrossConfirmFail(channelid string, txid string) error{

	//RollBack
	rollback(channelid, txid)
	//此跨链tx相关内存变量删除
	processCrossVar(txid)
	//跨链数据库更新为失败
	processCrossDB(txid, "Mult-Fail")

	return nil
}
*/

// 协同跨链确认成功
func CrossConfirmSucc(txid string) error{
	//此跨链tx相关内存变量删除（解锁）
	processCrossVar(txid)
	//跨链数据库更新成功
	processCrossDB(txid, "Mult-Succ")

	return nil
}

//协同跨链确认失败
//func CrossConfirmFail(channelid string, txid string) error{
func CrossConfirmFail(txid string) error{
	//RollBack
//	rollback(channelid, txid)
	//此跨链tx相关内存变量删除
	processCrossVar(txid)
	//跨链数据库更新为失败
	processCrossDB(txid, "Mult-Fail")

	return nil
}

// 删跨链相关变量，解锁
// 一个key只会被一个tx锁
// []string-CrossTxID LockedKeys; map[string][]string - CrossTxKeyMap CrossTxReadKeyMap; map[string][]KeyOrigVal - CrossTxWriteKeyMap
func processCrossVar(txid string) error{
	PrintMapsLists()
	fmt.Println("crossvalidate.go processCrossVar() 111")
	DelLockedKeyByTxID(txid)
	fmt.Println("crossvalidate.go processCrossVar() 222")
	DelMapsItem(txid)
	fmt.Println("crossvalidate.go processCrossVar() 333")
	DelCrossTxID(txid)
	fmt.Println("crossvalidate.go processCrossVar() 444")
	PrintMapsLists()
	return nil
}
/*
// 根据内存相应originalVal，回滚
// 只有写操作有originalVal
func rollback(channelid string, txid string) error{
	// get ledger
	//ledger, _ := ledgermgmt.GetLedger(channelid)
	//ledger := peer.GetLedger(channelid)  //返回的是 ledger.PeerLedger (interface)
	ledger := peer.GetLedgerForRollback(channelid)  //ledger类型 kvledger.RollbackInterface (interface)

	keyOrigVal := CrossTxWriteKeyMap[txid]  // 是一个数组

	ledger.CrossRollbackOrigVal(&keyOrigVal)

//getstate能马上get，但putstate只有到commit阶段才能put进去

	return nil
}*/

func GetRollbackKV(txid string) *[]statedb.KeyOrigVal{
	kv := CrossTxWriteKeyMap[txid]
	return &kv
}

// 跨链数据库每条数据有3种状态：单跨链成功、单跨链失败、多跨链未完成、多跨链成功、多跨链失败
func processCrossDB(txid string, crossStatus string) error{
//	updateCrossDBStatus(txid, crossStatus)
	return nil
}
