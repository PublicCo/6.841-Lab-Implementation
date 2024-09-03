## 实现参考

我参考了这个[链接](https://blog.imfing.com/2020/09/mit-6.824-lab1-map-reduce/)的实现思路。这个博客说的很详细了，把结构体定义出来就可以开始着手自己实现了。

我的具体实现思路已在代码中注释。

## 配置调试遇到的问题：插件构建出错

出现这个问题是因为debugger要求debug前所有内容都要是正确的。可以查看这个[链接](https://stackoverflow.com/questions/70642618/cannot-load-plugin-when-debugging-golang-file-in-vscode)或者这个[链接](https://juejin.cn/post/7211861991533690937)

需要额外添加一个指令：-gcflags "all=-N -l"

你可以参照我在.vscode下的launch.json和task.json进行配置。

要注意：构建插件和go run版本要一致，要同时开关-race，否则会出现插件版本不对的报错

## Struct大小写问题

喜闻乐见的是，如果struct内成员是小写是不可导出的

虽然你的代码不会出问题（因为都在包内），但是进行Call通信的时候就没办法传输了

更可怕的是，Hint中指出的”静默返回错误的值”指的是空值（或者所有是错误命名的值）

除了reply结构体返回的是空值，其他完全正常，ok都是返回true。

**所以务必记住，rpc里的所有成员都应该是大写字母开头（可导出的）**

## 超时重分配问题

不知道是不是电脑不行，三线程运行状态下每个线程将一个文件map拆开需要超过20秒

所以要求十秒重分配会导致反复重分配进而导致死循环

可以强制提高阈值



## 关于Tempfile的问题

os.createtemp不能使用默认路径“”（如果你和我一样用的是wsl），因为似乎程序是在Windows上跑的（mnt挂载在win系统），但是tmpfile是在var/tmp。这会导致os.rename出错：报错不在同一个文件系统上

## 关于测试问题

经过仔细的调查，发现我的cpu跑不动这么多个线程，导致运行一次需要跑个几分钟。而运行最多允许跑120秒。这导致我还在输出中间文件，脚本就把我掐了

具体方法是找到这段

~~~bash
if [ "$TIMEOUT" != "" ]
then
  TIMEOUT2=$TIMEOUT
  TIMEOUT2+=" -k 2s 1200s "
  TIMEOUT+=" -k 2s 450s "
fi
~~~

让timeout长点，别切那么快

如果电脑顶不住，可以在下面的bash命令里少开几个线程，不然coordinator会不停地超时重传

另外脚本调不对，如果逐步输入脚本命令是能正确运行的，但是脚本不行

可能这就是wsl下人一等吧（悲）

**2024.9.1 该问题已解决**：出现该问题的原因是使用mnt挂载win系统并在该系统下运行程序会导致读写操作奇慢无比（毕竟一直在外部设备中读写）。你需要将相关文件挂在在wsl的Linux系统下（我现在挂在在home下），可以显著提高读写速度（上百倍）并解决Rename冲突

![image-20240901100056611](C:\Users\leon\AppData\Roaming\Typora\typora-user-images\image-20240901100056611.png)

圆满完成！