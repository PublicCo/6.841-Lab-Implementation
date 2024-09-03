package kvsrv

type AppendState int

const (
	Modify AppendState = iota
	Ack
)

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.

	// 为了确保每个操作幂等，需要给每个操作一个唯一ID
	ID int64

	// 根据Hint3，需要设计一个syn-ack机制，使得客户端通知服务端清除id缓存
	State AppendState
}

type PutAppendReply struct {
	Value string
}

type GetArgs struct {
	Key string
	// You'll have to add definitions here.
}

type GetReply struct {
	Value string
}
