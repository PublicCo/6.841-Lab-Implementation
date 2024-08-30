package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Coordinator struct {
	// Your definitions here.
	mut sync.Mutex

	filename []string
	nreduce  int

	maptask    []Task
	reducetask []Task

	maptaskfinish    bool
	reducetaskfinish bool
}

// Your code here -- RPC handlers for the worker to call.

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
//	func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
//		reply.Y = args.X + 1
//		return nil
//	}
func (c *Coordinator) Example(args *CallArgs, reply *TaskReply) error {
	tmp := TaskReply{}
	tmp.Task.Index = 99
	reply.Task = tmp.Task
	return nil
}

// 响应Map完成事件
func (c *Coordinator) HandleMapFinish(args *CallArgs, reply *TaskReply) error {
	c.mut.Lock()
	defer c.mut.Unlock()
	// 检测时间戳
	if args.Task.Timestamp == c.maptask[args.Task.Index].Timestamp {
		// 更改task状态
		c.maptask[args.Task.Index].Missionstate = Done
	}

	return nil
}

// 分配Map任务给Worker
func (c *Coordinator) AssignMission(args *CallArgs, reply *TaskReply) error {
	c.mut.Lock()
	defer c.mut.Unlock()
	tmpCheckAllFinish := true // 检测是否全完成了。如果是false说明至少有一个任务ready或running
	reply.NReduce = c.nreduce // 传输nreduce个数，确保worker可以用到
	// 首先完成map任务分配
	if !c.maptaskfinish {
		i := 0
		for ; i < len(c.maptask); i++ {
			currenttime := time.Now()

			// 如果任务未分配或者任务超过了10秒
			if (c.maptask[i].Missionstate == Ready) || (c.maptask[i].Missionstate == Running && currenttime.Sub(c.maptask[i].Timestamp).Seconds() > 10) {
				// 重新分配
				c.maptask[i].Timestamp = time.Now() //更新时间戳
				c.maptask[i].Missionstate = Running
				reply.Task = c.maptask[i]

				// 分配任务成功，说明map任务未完成
				c.maptaskfinish = false

				// 完成，返回给worker执行
				return nil
			} else if c.maptask[i].Missionstate == Running {
				// 还有任务正在运行，map任务没有结束
				tmpCheckAllFinish = false
			}
		}
		// 遍历了一遍，可以确认是否完成了
		c.maptaskfinish = tmpCheckAllFinish

		//如果运行到这里，说明没有找到一个合适的任务进行分配。因此分配为waiting
		assert(i == len(c.maptask), "任务分配出错：未遍历完成任务列表即跳出")
		reply.Task.Worktype = Waiting
		return nil
	} else {
		// 完成reduce任务分配
	}

	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	ret := false
	c.mut.Lock()
	defer c.mut.Unlock()
	// Your code here.

	// 如果mapfinish和reducefinish就Done
	if c.maptaskfinish && c.reducetaskfinish {
		ret = true
	}
	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}
	// 第一步，初始化C的数据结构
	// 为C设定nReduce个数
	c.nreduce = nReduce

	//初始化要分配的文件名
	for _, possiblefile := range files {
		files, err := filepath.Glob(possiblefile)
		if err != nil {
			fmt.Println("错误读取文件：", err)
			return nil
		}
		c.filename = append(c.filename, files...)
	}

	//初始化可以分配的maptask
	for i, filename := range c.filename {
		tmp := Task{}
		tmp.Missionstate = Ready
		tmp.Index = i
		tmp.Mapfile = filename
		c.maptask = append(c.maptask, tmp)

	}

	// 初始化reducetask
	for i := 0; i < c.nreduce; i++ {
		c.reducetask = append(c.reducetask, Task{})
		c.reducetask[i].Worktype = Reduce
		c.reducetask[i].Index = i
		c.reducetask[i].Missionstate = Ready

		// 初始化每个要读取的reducefile的文件名
		for index := range c.filename {
			c.reducetask[i].Reducefiles = append(c.reducetask[i].Reducefiles, fmt.Sprintf("mr-%v-%v", index, i))
		}
	}

	c.server()
	return &c
}
