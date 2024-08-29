package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

// main/mrworker.go calls this function.
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	// 记录传输args和得到coordinator的reply
	for {
		reply := TaskReply{}
		args := CallArgs{}
		worktype := CallForMapMission(&args, &reply)
		switch worktype {
		case Waiting:
			time.Sleep(2 * time.Second)
		case Map:
			HandleMap(&reply, mapf)
		}
	}

}

/*TODO: 在worker进行map/reduce的时候，完成任务时需要保存文件。
保存完之后需要将文件名发送给coordinator.因此需要额外写一个处理函数*/

// 进行map操作然后将完成的文件发送给coordinator
func HandleMap(reply *TaskReply, mapf func(string, string) []KeyValue) {
	// 读取file并拆出中间键

	filename := reply.task.mapfile
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("cannot open %v", filename)
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatalf("cannot read %v", filename)
	}
	file.Close()
	kva := mapf(filename, string(content))

	// 创建nreduce个临时文件准备写入
	var ReduceFileList []*os.File      //用来记录每个临时文件的句柄避免句柄泄露
	var ReduceJsonList []*json.Encoder // 记录json编码器方便后续kv写入
	for i := 0; i < reply.NReduce; i++ {
		// 创建nreduce个临时文件
		TempFilename := fmt.Sprintf("Tempfile-%d-%d", reply.task.index, i) // 第一个参数是TaskId，第二个是第几个reduce
		tmpfile, err := os.CreateTemp("", TempFilename)
		if err != nil {
			log.Fatalf("Cannot Open %v", TempFilename)
		}

		// 为每个tmpfile创建json编码器，方便编码解码用
		enc := json.NewEncoder(tmpfile)
		ReduceFileList = append(ReduceFileList, tmpfile)
		ReduceJsonList = append(ReduceJsonList, enc)
	}
	// 释放句柄
	defer func() {
		for _, file := range ReduceFileList {
			err := file.Close()
			if err != nil {
				log.Print("Error closing the file", err)
			}
		}
	}()

	//对于kva，我们将每个键进行ihash%nreduce,将它写入指定的临时文件里
	for _, onekv := range kva {
		ReduceBucketNum := ihash(onekv.Key) % reply.NReduce

		//我们使用Json格式将kv写入文件里
		err := ReduceJsonList[ReduceBucketNum].Encode(&onekv)
		if err != nil {
			log.Fatal("Unable to Write Json into the file", err)
		}

	}

	// 完成后，原子性地重命名原文件，确保结果文件不存在崩溃
	for i, file := range ReduceFileList {
		err := os.Rename(file.Name(), fmt.Sprintf("mr-%v-%v", reply.task.index, i))
		if err != nil {
			log.Fatal("Unable to Rename the Result File", err)
		}

	}
	//向Coordinator发送完成通知预备数组结构
	TempReply := TaskReply{} // 完成后该进程不需要获取reply，入参用，完成call后丢弃
	FinishArgs := CallArgs{}

	FinishArgs.CallType = ReturnTask         // Call类型为完成
	FinishArgs.task.index = reply.task.index //返回完成的任务序号
	FinishArgs.task.timestamp = reply.task.timestamp
	// 由于coordinator知道taskid和nreduce，因此可以自己编码相关文件名，通知一声即可
	ok := call("Coordinator.HandleMapFinish", &FinishArgs, &TempReply)
	if !ok {
		// 不能fatal，不ok说明暂时无法连上coordinator，log后退出即可
		log.Printf("HandleMap:Unable to coonect the coordinator")
	}
}

// Call for Mission
func CallForMapMission(args *CallArgs, reply *TaskReply) WorkType {
	//只需要将状态设置为等待任务即可，Task由master来修改assign
	args.CallType = RequestTask

	ok := call("Coordinator.AssignMission", args, reply)
	if !ok {
		log.Println("Call Mission:Unable to connect the coordinator")
		//无法连接服务器，等待后重新call
		reply.task.worktype = Waiting
	}
	return reply.task.worktype

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	// the "Coordinator.Example" tells the
	// receiving server that we'd like to call
	// the Example() method of struct Coordinator.
	ok := call("Coordinator.Example", &args, &reply)
	if ok {
		// reply.Y should be 100.
		fmt.Printf("reply.Y %v\n", reply.Y)
	} else {
		fmt.Printf("call failed!\n")
	}
}

// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
