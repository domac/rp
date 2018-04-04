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
rp.CreateProfile()
```
使用配置文件

```
rp.LoadDebugProfile("/path/to/config/file")

rp.StartProfile(10029,"/tmp/prof.cpu", "/tmp/prof.mem", 30*time.Second)
```

如使用配置,则根据配置的项目名定义调用的端口,主要用于单台机器上启动多个profile服务的场景

配置文件格式:

```
[debug_profile]
module_ports = [7000,7001,7002]
module_names = ["test","ppdemo","domac"]
profile_output_dir = "../ppdemo/pdata"
```

### 快照文件检测例子：

程序运行过程中，调用 rp 的开发api

```
curl http://localhost:10029
```

上面的请求会同时产生

所有的性能profile文件,若只想生成特定的性能快照,可以参考如下:

```
生成CPU快照
curl http://localhost:10029?mode=1

生成MOMERY快照
curl http://localhost:10029?mode=2

生成BLOCK快照
curl http://localhost:10029?mode=3

生成TRACE快照
curl http://localhost:10029?mode=4
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
