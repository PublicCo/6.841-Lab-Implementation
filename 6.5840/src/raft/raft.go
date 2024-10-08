package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"

	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type ServerState int

const (
	Follower ServerState = iota
	Candidate
	Leader
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers，记录每个server的网络标识符，用于向它们通信
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	state       ServerState //描述是candidate，leader还是follower
	currentTerm int         //描述本身server中log最新的版本号
	votedFor    int         //记录该服务器投给某个节点leader，该leader的index
	votednumber int         // 在candidate选举中记录支持的人数。
	// 不在论文内的参数
	lastTimeReceiveRPC time.Time //描述上一次获得RPC是什么时候，用于检测是否超时
}

// Log传输数据结构
type AppendEntries struct {
	Term         int // leader 当前的版本号
	LeaderID     int // leader 的index（唯一ID）
	PrevlogIndex int // 上一个log entry的index
	PrevlogTerm  int // 上一个log entry所属的任期
	/*entries	要添加的日志
	leaderCommit leader要将该日志commit到哪一个位置
	*/
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isleader bool
	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	term = rf.currentTerm
	isleader = (rf.state == Leader)
	return term, isleader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).

}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	// 根据论文figure2 填写需要的args
	Term         int //candidate的版本
	CandidateId  int //candidate的index（唯一id）
	LastLogIndex int //candidate上一次log entry的序号，判断是否正确
	LastLogTerm  int //candidate上一次log entry是由哪个版本的leader发出的
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	// 根据论文figure2 填写需要的result
	Term        int  //follower当前的term。如果candidate的term没有follower早的话应该更新
	VoteGranted bool //是否同意让它当选
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	// 实现的是每个follower返回给candidate的结果
	rf.mu.Lock()
	defer rf.mu.Unlock()
	reply.Term = rf.currentTerm
	//如果任期比follower还旧，return false
	if args.Term < reply.Term {
		reply.VoteGranted = false

	} else {
		// 每个server在接收到服务器版本比自己高后，都应该更新自己的版本号
		if rf.currentTerm < args.Term {
			// DPrintf("Server %v term is %v,I am %v,my term is %v", args.CandidateId, args.Term, rf.me, rf.currentTerm)
			rf.currentTerm = args.Term
			// 自己的版本已经比最新的版本要旧了，因此需要重置当前版本的leader
			rf.votedFor = -1
			//自己已经过时了，变回follower
			rf.state = Follower
		}
		//如果自己已经vote其他人了，不能继续vote
		//TODO：需要检查两个日志中最后一次log的版本是否candidate比自己新
		if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
			// TODO:in lab B
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
			// DPrintf("Server %v accept Server %v to be leader", rf.me, args.CandidateId)

		} else {
			reply.VoteGranted = false
		}
	}
	// 投票后reset 定时器
	rf.lastTimeReceiveRPC = time.Now()
}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (3B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here (3A)
		// Check if a leader election should be started.
		// pause for a random amount of time between 200 and 400
		// milliseconds.
		startWaitTime := time.Now()
		// 先等一等，然后发起选举请求
		ms := 300 + (rand.Int63() % 200)
		time.Sleep(time.Duration(ms) * time.Millisecond)

		// 如果在上一次wait之后一直没有收到消息，请求选举
		rf.mu.Lock() //访问rf数据，需要上锁
		if rf.lastTimeReceiveRPC.Before(startWaitTime) {
			// 如果rf本身已经是leader了，有没有收到消息不打紧
			if rf.state != Leader {
				// DPrintf("Server %v start to elect", rf.me)
				go rf.emitElection()
			}
		}
		rf.lastTimeReceiveRPC = startWaitTime
		rf.mu.Unlock()
	}
}

func (rf *Raft) emitElection() {
	rf.mu.Lock()

	// 首先增加自己的term
	rf.currentTerm++
	// 切换成candidate状态
	rf.state = Candidate
	// 切换votefor

	rf.votedFor = rf.me
	rf.votednumber = 1
	args := RequestVoteArgs{
		Term:        rf.currentTerm,
		CandidateId: rf.me,
	}
	rf.mu.Unlock()

	//peers数组大小应当是固定的，因为lab没有要求实现添加server
	for i := range rf.peers {
		//循环内容需要go并发。此外，需要设定一段时间，超过这个时间认为选举失败
		//自己已经给自己投一票了
		if i == rf.me {
			continue
		}
		// 向所有server发送选举请求
		go func(i int, args RequestVoteArgs) {

			reply := RequestVoteReply{}
			//// DPrintf("Sending vote request to Server %v", i)
			ok := rf.sendRequestVote(i, &args, &reply)
			if ok && reply.VoteGranted {
				rf.mu.Lock()
				rf.votednumber++
				if rf.state == Candidate && rf.votednumber > len(rf.peers)/2 {
					rf.state = Leader
					rf.LeaderOperation()
				}
				rf.mu.Unlock()
			}
		}(i, args)

	}

}

// 作为leader要执行的函数
func (rf *Raft) LeaderOperation() {
	// DPrintf("Server %v become leader", rf.me)
	go rf.sendHeartBeat()
}

type HeartbeatReply struct {
	Term    int  // 自己的term
	Success bool // 是否成功更新
}

func (rf *Raft) sendHeartBeat() {
	const heartbeattime = 100
	for {
		// 循环向每个服务器发送log entries
		rf.mu.Lock()
		appendentries := AppendEntries{
			Term:     rf.currentTerm,
			LeaderID: rf.me,
		}
		rf.mu.Unlock()
		for i := range rf.peers {
			// 不用给自己发heartbeat
			if i == rf.me {
				continue
			}
			go func(i int, appendentries AppendEntries) {
				heartbeatreply := HeartbeatReply{}
				// 之后可能要处理这个ok
				rf.peers[i].Call("Raft.GetHeartbeat", &appendentries, &heartbeatreply)
				// 如果响应的term中由更大的term，说明leader应当被降级为follower
				rf.mu.Lock()
				// 应当是判断term，因为success可能关乎于是否成功链接
				if rf.currentTerm < heartbeatreply.Term {
					rf.state = Follower
					// DPrintf("Server %v is invalid to be a leader,he became follower due to connection with follower %v", rf.me, i)
				}
				rf.mu.Unlock()

			}(i, appendentries)
			//当前leader不合法，自杀
			if rf.state != Leader {
				// DPrintf("Server %v is no longer a leader,Down", rf.me)
				break
			}
		}
		if rf.state != Leader {
			break
		}
		time.Sleep(time.Duration(heartbeattime) * time.Millisecond)
	}
}

// follower接收到heartbeat后进行处理
func (rf *Raft) GetHeartbeat(appendentries *AppendEntries, heartbeatreply *HeartbeatReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// 如果自己的版本比leader的版本大，reply false
	if rf.currentTerm > appendentries.Term {
		heartbeatreply.Success = false
		// DPrintf("Fail to accept heartbeat: Leader %v term is older than Me %v", appendentries.LeaderID, rf.me)
		// elif ...
	} else {
		rf.state = Follower
		rf.currentTerm = appendentries.Term
		// 更新RPC收取时间戳
		// DPrintf("Server %v get heartbeat from Leader %v", rf.me, appendentries.LeaderID)
		rf.lastTimeReceiveRPC = time.Now()
		heartbeatreply.Success = true
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (3A, 3B, 3C).

	// 初始化其他有必要的参数
	rf.state = Follower // 刚启动，先设定follower
	rf.currentTerm = 0  //尚未获取，先为0
	rf.votedFor = -1    //还没有进行选举，设定为不存在
	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
