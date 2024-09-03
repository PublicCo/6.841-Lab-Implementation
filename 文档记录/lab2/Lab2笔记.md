这次的任务主要难点在于解决put的幂等性问题：在客户端多次put的情况下如何确保其相当于只put了一次。一个可能的错误是：

现在有机子1和2，正确的写是1 put 2 put，但是1发现put错误了（没有正确发送）就会重新发送一次put变成1 put 2 put 1 put，这样文件就变成1的内容了（本来应该是2的内容）。我们要确保无论1 重发多少次，只要接收到一次完成修改，其他的消息都会无视

* 对于get操作，它本身是幂等的，因此只需要不停地发送即可
* 对于put操作，需要用唯一id标识记录该操作。如果id已经存在，说明已经执行过了，直接跳过执行
  * 需要一个额外的ack操作来向client确认已经完成了put操作，避免id在server中堆积太多

## 问题说明

最大的问题是本题的需求。因为test主要检查你的返回值是否正确，因此在这里重新梳理返回值的要求

* 对于get方法，返回完成获取的最新的数据。你不需要关心get前kv服务器是什么状态。
* 对于put/append方法：**返回在完成该操作之前，map[key]的内容**
  * 例如，服务器已有map["1"]="apple"了，在第一次传输put("1","banana")后，reply.Value应当返回“apple”。在这之后只要put操作的id相同，reply都应该返回“apple”

唯一的卡点是返回值，其他都很简单。

一个简单的脚本配置文件如下：

~~~
{
            "name": "Lab2 Debugger",
            "type": "go",
            "request": "launch",
            "mode": "test",
            "program": "${workspaceFolder}/src/kvsrv",
            "args": [

            ],
            "buildFlags": ["-race"],
            "showLog": true,
        },
~~~

![image-20240903142328622](C:\Users\leon\AppData\Roaming\Typora\typora-user-images\image-20240903142328622.png)
