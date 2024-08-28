package mr

import (
	"fmt"
	"hash/fnv"
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

		}
	}

}

/*TODO: 在worker进行map/reduce的时候，完成任务时需要保存文件。
保存完之后需要将文件名发送给coordinator.因此需要额外写一个处理函数*/
// Call for Mission
func CallForMapMission(args *CallArgs, reply *TaskReply) WorkType {
	//只需要将状态设置为等待任务即可，Task由master来修改assign
	args.CallType = RequestTask

	ok := call("Coordinator.AssignMission", args, reply)
	if !ok {
		log.Println("Unable to connect the coordinator,the worker down")
		os.Exit(-1)
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
