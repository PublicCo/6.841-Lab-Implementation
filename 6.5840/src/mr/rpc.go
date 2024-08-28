package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import (
	"log"
	"os"
	"strconv"
	"time"
)

//
// example to show how to declare the arguments
// and reply for an RPC.
//

// Self Defined Assert
func assert(condition bool, message string) {
	if !condition {
		log.Println(message)
		os.Exit(-1)
	}
}

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// 描述当前任务状态
type MissionState int

const (
	Ready MissionState = iota
	Running
	Done
)

// 描述任务要做map还是reduce还是还没assign
type WorkType int

const (
	Map WorkType = iota
	Reduce
	Waiting
)

type Task struct {
	worktype     WorkType     // 当前task在被map还是reduce
	missionstate MissionState // 当前Task有没有被分配
	index        int          // 在task数组中的哪一个地方
	timestamp    time.Time    //十秒钟重新分配
	mapfile      string       //Map的单位是一个file
	reducefiles  []string     //十个reducer，每个分配不同数量的files
}

// worker要发给coordinator有两件事：请求文件，任务完成返回文件名
type WorkerMessageType int

const (
	RequestTask WorkerMessageType = iota //请求任务
	ReturnTask                           //完成任务返回内容
)

type CallArgs struct {
	CallType WorkerMessageType //任务类型
	task     Task              //任务数据结构
}

type TaskReply struct {
	task    Task //任务数据结构
	NReduce int  //在reduce中要完成哪一块的reduce任务
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/5840-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
