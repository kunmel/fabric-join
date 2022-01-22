package cross

import (
	"fmt"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
)

// peer跨链共维护5个表：list-2, map-3
var CrossTxID = make([]string, 0, 10)
var LockedKeys = make([]string, 0, 10)

/*type KeyOrigVal struct{
	key	string
	originalVersionedValue  []byte
}*/
/*
type KeyOrigVal struct{
	Namespace              string
	Key                    string
	OriginalVersionedValue *statedb.VersionedValue
}*/
// txID : []KeyOrigVal   这个txID是本地的txID，再加一个crossID:txID   //只有PutState维护
//var CrossTxKeyOrigValMap = make(map[string][]KeyOrigVal)  
var CrossTxWriteKeyMap = make(map[string][]statedb.KeyOrigVal)
// txID : []string(key)	 只有GetState维护  一个txID可以对应多个read key
var CrossTxReadKeyMap = make(map[string][]string)
// txID : []string(key)  该txID对应的所有key
var CrossTxKeyMap = make(map[string][]string)

func PrintMapsLists(){
	fmt.Println("打印所有List Map")
	fmt.Println(CrossTxID)
	fmt.Println(LockedKeys)
	fmt.Println(CrossTxWriteKeyMap)
	fmt.Println(CrossTxReadKeyMap)
	fmt.Println(CrossTxKeyMap)
	fmt.Println("打印结束")
}

// is item in array? return item's index
func Contains(array []string, item string) (index int){
	index = -1
	if len(array) == 0{
		return	//数组空
	}else{
		for i:=0; i<len(array); i++{
			if array[i] == item{
				index = i
				return	//存在，返回index	
			}		
		}
	}
	return //不存在
}

// is key in LockedKeys[]string? in-true; notin-false
func IsLocked(key string) bool{
	index := Contains(LockedKeys, key)
	if index == -1{
		return false	
	}
	return true
}


// is this txid a crossTx?
func IsCrossTx(txid string) bool{
	index := Contains(CrossTxID, txid)
	if index == -1{
		return false		
	}
	return true
}

//	return isCross, isLocked   是跨链tx吗？是locked key吗？
func IsCrossTxLockedKey(txid string, key string) (bool, bool) {
	var cross, lock bool
	if cross = IsCrossTx(txid); cross{//如果是跨链tx,
		if lock = IsLocked(key); lock{ //还要判断是不是本tx锁的这个key, y-不算锁；n-lock
			if _, ok := CrossTxKeyMap[txid]; !ok {//如果这条尚不存在，说明一定是其他tx锁的key
				return true, true
			}	
	
			if index := Contains(CrossTxKeyMap[txid], key); index != -1{//CrossTxKeyMap[txid]是本tx目前已有的锁key，不是本tx锁的key
				return true, true
			}else{ //是这个tx锁的，不算
				return true, false	
			}
		}else{ //不在LockedKeys里，肯定没锁
			return true, false
		}
	}else { //不是跨链tx
		if lock = IsLocked(key); lock{ //该key已锁
			return false, true		
		}else{ //该key未锁
			return false, false 
		}
	}
}
//
func AppendCrossTxID(id string){
//	fmt.Println("peer/cross/cross.go AppendCrossTxID  id =", id)
	res := Contains(CrossTxID, id)
	if res > -1{ //有
		return
	}
	CrossTxID = append(CrossTxID, id)
}

//
func AppendLockedKey(key string){
	res := Contains(LockedKeys, key) //是否存在
	if res == -1{
	//不存在key
		LockedKeys = append(LockedKeys, key)
	}
}


//
/*func AppendWriteKeyMap(txID string, ns string, k string, v *statedb.VersionedValue){
	if _, ok := CrossTxWriteKeyMap[txID]; !ok {
	//txID不存在，初始化一个	
		slice := make([]statedb.KeyOrigVal, 0, 0)
		CrossTxWriteKeyMap[txID] = slice
	}
	//Key OriginalValue append to map.value
	kOrigV := statedb.KeyOrigVal{Namespace: ns, Key: k, OriginalVersionedValue: v}
	CrossTxWriteKeyMap[txID] = append(CrossTxWriteKeyMap[txID], kOrigV)

	//append this key to txIDKey map
	AppendKeyMap(txID, k)
}*/
func AppendWriteKeyMap(txID string, ns string, k string, v []byte){
	if _, ok := CrossTxWriteKeyMap[txID]; !ok {
		//txID不存在，初始化一个
		slice := make([]statedb.KeyOrigVal, 0, 0)
		CrossTxWriteKeyMap[txID] = slice
	}
	//Key OriginalValue append to map.value
	kOrigV := statedb.KeyOrigVal{Namespace: ns, Key: k, OriginalVersionedValue: v}
	CrossTxWriteKeyMap[txID] = append(CrossTxWriteKeyMap[txID], kOrigV)

	//append this key to txIDKey map
	AppendKeyMap(txID, k)
}

//
func AppendReadKeyMap(txID string, k string){
	if _, ok := CrossTxReadKeyMap[txID]; !ok {
	//txID不存在，初始化一个	
		slice := make([]string, 0, 0)
		CrossTxReadKeyMap[txID] = slice
	}
	CrossTxReadKeyMap[txID] = append(CrossTxReadKeyMap[txID], k) //每个txid对应一个读key数组

	//append this key to txIDKey map
	AppendKeyMap(txID, k)
}

//
func AppendKeyMap(txID string, k string){
	if _, ok := CrossTxKeyMap[txID]; !ok {
	//txID不存在，初始化一个	
		slice := make([]string, 0, 0)
		CrossTxKeyMap[txID] = slice
	}
	CrossTxKeyMap[txID] = append(CrossTxKeyMap[txID], k) //每个txid对应一个读key数组
}

// 根据CrossTxKeyMap
func DelLockedKeyByTxID(txid string){
	keys := CrossTxKeyMap[txid]
	for _, k := range keys{
		DelLockedKey(k)
	}
}
//
func DelLockedKey(key string){
	for index, item := range LockedKeys{
		if item == key{
			LockedKeys = append(LockedKeys[:index], LockedKeys[index+1:]...)
		}
	}
}
func DelMapsItem(txID string){
	delete(CrossTxKeyMap, txID)
	delete(CrossTxWriteKeyMap, txID)
	delete(CrossTxReadKeyMap, txID)
}
// CrossTxID
func DelCrossTxID(txid string){
	for index, item := range CrossTxID{
		if item == txid{
			CrossTxID = append(CrossTxID[:index], CrossTxID[index+1:]...)
		}
	}
}
/*
// txid{key:originalValue}
// if tx is a crossTx Read, maintain slice LockedKey
func PreCrossTxR(txID string, key string){
	if yes := IsCrossTx(txID); yes{ //如果是跨链tx
		AppendLockedKey(key)
//		AppendIdKeyMap(txID, key)
		AppendReadKeyMap(txID, key)
	}
}
// if tx is a crossTx Write, maintain slice LockedKey, CrossTxKeyOrigValMap
func PreCrossTxW(txID string, ns string, key string, v *statedb.VersionedValue){
	if yes := IsCrossTx(txID); yes{ //如果是跨链tx  /////////!!!!!!!!!!!!!!!!!不必要
		AppendLockedKey(key)
//		AppendIdKeyOrigValMap(txID, key, v)
		AppendWriteKeyMap(txID, ns, key, v)
	}
}
*/
// txid{key:originalValue}
// if tx is a crossTx Read, maintain slice LockedKey
func PreCrossTxR(txID string, key string){
//在core/endorser/endorser.go ProcessProposal()中，先用CrossTxID判断是不是跨链，再决定用不用这个函数。 所以不用判断跨链了
	AppendLockedKey(key)
	//		AppendIdKeyMap(txID, key)
	AppendReadKeyMap(txID, key)
}
// if tx is a crossTx Write, maintain slice LockedKey, CrossTxKeyOrigValMap
/*func PreCrossTxW(txID string, ns string, key string, v *statedb.VersionedValue){
	AppendLockedKey(key)
	//		AppendIdKeyOrigValMap(txID, key, v)
	AppendWriteKeyMap(txID, ns, key, v)
}*/
func PreCrossTxW(txID string, ns string, key string, v []byte){
	AppendLockedKey(key)
	//		AppendIdKeyOrigValMap(txID, key, v)
	AppendWriteKeyMap(txID, ns, key, v)
}