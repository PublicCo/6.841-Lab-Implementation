package kvsrv

import (
	"log"
	"sync"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

// 存储ID是否存在的struct。值不重要，因此空就行

type KVServer struct {
	mu sync.Mutex

	// Your definitions here.
	StoreMap map[string]string
	IDMap    map[int64]string
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	// 获取键值对
	value, exist := kv.StoreMap[args.Key]
	if exist {
		reply.Value = value
	} else {
		reply.Value = ""
	}
}

// 检查是否需要进行插入操作
// 如果返回true，那么需要进行put或者append
// 否则返回false，并清除id（如果state是ack）
func (kv *KVServer) CheckIfNeedChange(args *PutAppendArgs) bool {
	// 清除缓存
	if args.State == Ack {
		delete(kv.IDMap, args.ID)
		return false
	}

	// 检查是否已完成
	_, exist := kv.IDMap[args.ID]
	return !exist
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	// 检查是否已完成
	ok := kv.CheckIfNeedChange(args)
	if !ok {
		reply.Value = kv.IDMap[args.ID]
		return
	}
	//上一次操作的数据
	reply.Value = kv.StoreMap[args.Key]
	//修改操作
	kv.StoreMap[args.Key] = args.Value

	// 记录操作id
	kv.IDMap[args.ID] = args.Value

}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	kv.mu.Lock()
	defer kv.mu.Unlock()
	// 检查是否已完成
	ok := kv.CheckIfNeedChange(args)
	if !ok {
		reply.Value = kv.IDMap[args.ID]
		return
	}
	// 获取上一次的数据
	reply.Value = kv.StoreMap[args.Key]
	// 添加操作
	_, ok = kv.StoreMap[args.Key]
	if !ok {
		kv.StoreMap[args.Key] = args.Value
	} else {
		kv.StoreMap[args.Key] = reply.Value + args.Value
	}

	// 记录操作id
	kv.IDMap[args.ID] = reply.Value

}

func StartKVServer() *KVServer {
	kv := new(KVServer)

	// You may need initialization code here.

	// 初始化键值对map
	kv.StoreMap = make(map[string]string)

	// 初始化IDMap
	kv.IDMap = make(map[int64]string)
	return kv
}
