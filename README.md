# rp
runtime profile api

简单地对runtime profile生成方式进行封装，主要用于项目的debug

生成快照后，可以用pprof工具进行相关性能检测：

### 获取方式：

```
go get -u -v github.com/domac/rp
```

在相关项目代码中加入调用代码，例如：

```go
rp.StartProfile(10029,"/tmp/prof.cpu", "/tmp/prof.mem", 30*time.Second)

或直接

rp.CreateProfile(rp.MODE_DEBUG_PROFILE_CPU)
```

使用配置文件

```
rp.LoadDebugProfile("/path/to/config/file")

rp.StartProfile(10029,"/tmp/prof.cpu", "/tmp/prof.mem", 30*time.Second)
```

如使用配置,则根据配置的项目名定义调用的端口,主要用于单台机器上启动多个profile服务的场景

### 快照文件检测例子：

程序运行过程中，调用 rp 的开发api

```
curl http://localhost:10029
```

调用结束后，会生成相关的快照文件,可以通过pprof工具进行检测

> 在使用官方的pprof前, 请先在安装 [Graphviz](https://www.graphviz.org/download/)

pprof 获取方式：

```
go get -u -v https://github.com/google/pprof
```

使用例子：

```
pprof -http=:8000 prof.cpu
```

![p1](http://og0usnhfv.bkt.clouddn.com/p1.png)

![p2](http://og0usnhfv.bkt.clouddn.com/p2.png)
