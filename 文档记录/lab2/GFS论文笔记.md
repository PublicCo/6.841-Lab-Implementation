论文阅读参考了这篇[博客](http://hecenjie.cn/2020/01/31/%E3%80%8AGoogle-File-System%E3%80%8B%E8%AE%BA%E6%96%87%E7%AC%94%E8%AE%B0/),当然这个博客有些地方说的不是很明白，可以重新回到论文里看

一个论文翻译可以看[这里](https://kb.cnblogs.com/page/174130/)

## Abstract

总的来说，GFS是一个可扩展（？），承载大数据的分布式系统，并且有很好的容错能力。这个系统承载着几百TB的数据运行，能同时响应上百个客户端。

他是针对如下场景设计的

* 组件易失效
* 大文件存储

## Design Overview

### 问题假设

* 这个系统用的机器都很便宜，因此挂掉是很常见的。因此系统要有很强的tolerance。
* 系统存储数据很大。规模大概是几百万个100MB以上的文件
* 系统的读操作有两种：大规模流读或者小规模随机读（也就是说，很少存在读的量大且随机读的操作）大规模的量级是MB，小规模量级是KB
* 系统的写操作存在大量级顺序写操作（MB级）。一般没有覆盖写（修改文件），写了一般就不改了。小规模写操作没有这么多限制，但是对性能要求不高
* 系统要求文件同步。该系统用Producer-Consumer算法很多，而生产者可能是几百台机器。他们的文件要求全部一致。允许文件晚点被读到，但是读的内容是相同的

* 可以接受延迟高一点。使用者一般对读写响应速度要求不高，但是要以高速度批量处理数据

### 接口

实现了open，close，crud接口（create，delete，read，write）

此外，GFS还实现了快照和record append。 record append是每个client都可以同时写入，而不会出现死锁（也就是每个人都可以写，不会出现一个人连锁带写权限一起拿了不让其他人写）

### 系统架构

![image-20240901102921871](C:\Users\leon\AppData\Roaming\Typora\typora-user-images\image-20240901102921871.png)

client：用户端

chunkserver：存数据的主机。每一个都是一台Linux主机

master：管理文件系统

每个文件会被划分为固定大小的块（chunk），每个块都被一个64位 块句柄（chunk handle）标识（有点像指针地址）。handle是由master分配的。为了可靠，每个chunk会被存3个副本。

master维护文件系统。这个类似于Linux的文件系统，包括内存申请，块映射，块迁移等。主服务器通过heartbeat进行通信确认server存活

## 租约（lease）算法

* 实现目的：client在一次写操作过程不必要多次访问master（否则很多个client同时访问master会导致其成为访问热点）。该算法是通过将写入顺序管理权限下放至某个chunk中，让它成为临时的“小组长”，从而缓解写入压力
* 数据示意图如下：![image-20240902095611208](C:\Users\leon\AppData\Roaming\Typora\typora-user-images\image-20240902095611208.png)
* 算法流程如下
  1. client向master询问某个chunk的primary（小组长）server是谁。并获取该chunk的所有位置（存在哪个server的哪个目录下）
  2. master返回现有的primary chunkserver，如果没有就随机指定一个，并返回所有chunk副本位置
  3. client将要修改的chunk全部推送至每个server的缓冲区（控制流和数据流解耦）
  4. client向primary请求写入
  5. 由于可能几个client一起请求修改，primary会管理一个序列化的修改顺序，对文件进行修改。primary会将这个序列发给其他server
  6. 其他server按顺序修改并返回修改结果
  7. primary向client返回是否成功修改
* 几个细节
  * 对于租约，一般它的到期时间是60秒。然而，在文件正在修改时，租约可以无限续期。这个是通过心跳消息进行管理（primary需要一致报告自己还活着）。
  * 主服务器可以随时撤回租约。这可以让该primary server的修改无效（？应该是这样）。master也可能失去与primary server 的联络，不过没关系，租约到期后该primary server的写入顺序也作废了
    * 总的来说，首先是由主服务器来管理租约来选择primary server使用他的序列，然后primary server再按顺序通知其他server进行修改。

## 操作日志

* 这个是用来记录crud顺序，记录文件信息变更的内容，很重要。如果服务挂了，可以通过操作日志重演操作来恢复
* 当然，操作日志的数据量很大。因此master会时不时快照当前文件系统。出问题了就会从当前快照+快照后的操作日志反演回之前的文件系统。、
* **日志极其重要**，因此这一部分会被备份多份，在修改时会先将所有备份更新再返回client汇报。

## 数据流

对于chunkserver的网络拓扑，数据会沿着某一个路径进行推送。这个涉及到网络延迟侦测（一些计网算法会帮你解决对于某个节点serverS1，它应该向哪个方向推送实现最短路径节点全覆盖）

