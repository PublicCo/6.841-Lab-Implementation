package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"
)

// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// 自定义keyvalue的sort
// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

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
	//CallExample()

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
		case Reduce:
			HandleReduce(&reply, reducef)

		}
	}

}

func HandleReduce(reply *TaskReply, reducef func(string, []string) string) {
	// 读取file并取出中间键

	filelist := reply.Task.Reducefiles
	// 读取file
	kva := []KeyValue{}
	for _, filename := range filelist {
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}

		// 解码file，将kv读取进数组
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil {
				//EOF
				break
			}
			kva = append(kva, kv)
		}
	}
	// 论文里说你最好sort一下让key放在一起，那我们就sort一下
	sort.Sort(ByKey(kva))

	// 输出文件
	oname := fmt.Sprintf("mr-out-%v", reply.Task.Index)
	ofile, err := os.Create(oname)
	if err != nil {
		log.Fatalf("Unable to open the file %v", oname)
	}
	defer ofile.Close()
	TempFilename := "ReduceTempfile*"
	// 由于temp文件无法rename，需要保存在当前目录
	// 获取当前工作目录
	// currentDir, err := os.Getwd()
	// if err != nil {
	// 	fmt.Println("Error getting current directory:", err)
	// 	return
	// }
	tmpfile, err := ioutil.TempFile("", TempFilename)
	if err != nil {
		log.Fatalf("Cannot Open %v", TempFilename)
	}

	//按格式输出
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(tmpfile, "%v %v\n", kva[i].Key, output)

		i = j
	}

	// 原子化rename
	err = os.Rename(tmpfile.Name(), ofile.Name())
	if err != nil {
		log.Fatal("Unable to Rename the Result File:", err)
	}

	//向Coordinator发送完成通知预备数组结构
	TempReply := TaskReply{} // 完成后该进程不需要获取reply，入参用，完成call后丢弃
	FinishArgs := CallArgs{}

	FinishArgs.CallType = ReturnTask         // Call类型为完成
	FinishArgs.Task.Index = reply.Task.Index //返回完成的任务序号
	FinishArgs.Task.Timestamp = reply.Task.Timestamp
	// 由于coordinator知道taskid和nreduce，因此可以自己编码相关文件名，通知一声即可
	ok := call("Coordinator.HandleReduceFinish", &FinishArgs, &TempReply)
	if !ok {

		log.Fatalf("HandleReduceFinish:Unable to coonect the coordinator")
	}
	fmt.Printf("finish reduce task %v\n", reply.Task.Index)
}

// 进行map操作然后将完成的文件发送给coordinator
func HandleMap(reply *TaskReply, mapf func(string, string) []KeyValue) {
	// 读取file并拆出中间键

	filename := reply.Task.Mapfile
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

	// 由于temp文件无法rename，需要保存在当前目录
	// 获取当前工作目录
	// currentDir, err := os.Getwd()
	// if err != nil {
	// 	fmt.Println("Error getting current directory:", err)
	// 	return
	// }
	// 创建nreduce个临时文件准备写入
	var ReduceFileList []*os.File      //用来记录每个临时文件的句柄避免句柄泄露
	var ReduceJsonList []*json.Encoder // 记录json编码器方便后续kv写入
	for i := 0; i < reply.NReduce; i++ {
		// 创建nreduce个临时文件
		TempFilename := "Tempfile*"
		tmpfile, err := ioutil.TempFile("", TempFilename)
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
		outputfilename := fmt.Sprintf("mr-%v-%v", reply.Task.Index, i) // 第一个参数是TaskId，第二个是第几个reduce
		err := os.Rename(file.Name(), outputfilename)
		if err != nil {
			log.Fatal("Unable to Rename the Result File:", err)
		}

	}

	//向Coordinator发送完成通知预备数组结构
	TempReply := TaskReply{} // 完成后该进程不需要获取reply，入参用，完成call后丢弃
	FinishArgs := CallArgs{}

	FinishArgs.CallType = ReturnTask         // Call类型为完成
	FinishArgs.Task.Index = reply.Task.Index //返回完成的任务序号
	FinishArgs.Task.Timestamp = reply.Task.Timestamp
	// 由于coordinator知道taskid和nreduce，因此可以自己编码相关文件名，通知一声即可
	ok := call("Coordinator.HandleMapFinish", &FinishArgs, &TempReply)
	if !ok {

		log.Fatalf("HandleMap:Unable to coonect the coordinator")
	}
	fmt.Printf("finish task %v\n", reply.Task.Index)
}

// Call for Mission
// 因未知原因，疑似不能通过执行其他函数进行call。一切call函数都放在worker函数中
func CallForMapMission(args *CallArgs, reply *TaskReply) WorkType {
	//只需要将状态设置为等待任务即可，Task由master来修改assign
	args.CallType = RequestTask

	ok := call("Coordinator.AssignMission", args, reply)
	if !ok {
		log.Println("Call Mission:Unable to connect the coordinator")
		//无法连接服务器，等待后重新call
		reply.Task.Worktype = Waiting
	}

	return reply.Task.Worktype

}

// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
func CallExample() {

	// // declare an argument structure.
	// args := ExampleArgs{}

	// // fill in the argument(s).
	// args.X = 99

	// // declare a reply structure.
	// reply := ExampleReply{}

	// // send the RPC request, wait for the reply.
	// // the "Coordinator.Example" tells the
	// // receiving server that we'd like to call
	// // the Example() method of struct Coordinator.
	// ok := call("Coordinator.Example", &args, &reply)
	// if ok {
	// 	// reply.Y should be 100.
	// 	fmt.Printf("reply.Y %v\n", reply.Y)
	// } else {
	// 	fmt.Printf("call failed!\n")
	// }
	args := CallArgs{}
	args.Task.Index = 99
	reply := TaskReply{}
	ok := call("Coordinator.Example", &args, &reply)
	if !ok {
		fmt.Print("Error")
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
