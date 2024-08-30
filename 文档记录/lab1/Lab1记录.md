## 配置调试遇到的问题：插件构建出错

可以查看这个[链接](https://stackoverflow.com/questions/70642618/cannot-load-plugin-when-debugging-golang-file-in-vscode)或者这个[链接](https://juejin.cn/post/7211861991533690937)

需要额外添加一个指令：-gcflags "all=-N -l"

你在设计task.json来生成wc.so的时候应该有如下配置:

