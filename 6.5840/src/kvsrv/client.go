package kvsrv

import (
	"crypto/rand"
	"math/big"

	"6.5840/labrpc"
)

type Clerk struct {
	server *labrpc.ClientEnd
	// You will have to modify this struct.
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(server *labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.server = server
	// You'll have to add code here.
	return ck
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.

	// get方法本身是幂等的，如果没接收到只需要不停发送就可以了
	// 如果没有正确发送，客户端需要重复发送
	getargs := GetArgs{}
	getargs.Key = key
	getreply := GetReply{}
	for {
		ok := ck.server.Call("KVServer.Get", &getargs, &getreply)
		if ok {
			break
		}
	}

	return getreply.Value
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.server.Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) string {
	// You will have to modify this function.

	// 首先，对于每个操作都生成一个唯一ID
	// 由于randon函数都写好了，我们就随机分配ID。更好的方法应该是使用自增id
	id := nrand()
	args := PutAppendArgs{}
	args.Key = key
	args.Value = value
	args.ID = id
	args.State = Modify

	reply := PutAppendReply{}
	if op == "Put" {
		// 第一波循环，请求修改
		for {
			ok := ck.server.Call("KVServer.Put", &args, &reply)
			if ok {
				break
			}
		}
		reslt := reply.Value
		// 第二波循环，请求删除缓存
		args.State = Ack
		reply := PutAppendReply{}
		for {
			ok := ck.server.Call("KVServer.Put", &args, &reply)
			if ok {
				break
			}
		}
		return reslt
	} else { //op==append
		// 第一波循环，请求修改
		for {
			ok := ck.server.Call("KVServer.Append", &args, &reply)
			if ok {
				break
			}
		}
		reslt := reply.Value
		// 第二波循环，请求删除缓存
		args.State = Ack
		reply := PutAppendReply{}
		for {
			ok := ck.server.Call("KVServer.Append", &args, &reply)
			if ok {
				break
			}
		}
		return reslt
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}

// Append value to key's value and return that value
func (ck *Clerk) Append(key string, value string) string {
	return ck.PutAppend(key, value, "Append")
}
