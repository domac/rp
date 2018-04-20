# rp
服务运行时性能profile文件生成工具

对标准runtime profile生成方式进行封装，简化Go标准profile工具的使用方式，主要用于项目的debug

生成快照后，可以用[pprof工具](https://github.com/google/pprof)或命令`go tool pprof`进行相关性能检测：

### 工具使用：

```
go get -u -v github.com/domac/rp
```

在相关项目代码中加入调用代码，例如：

```go
rp.DEBUG_PROFILE()
```
使用配置文件

```
rp.LoadConfigFile("/path/to/config/file")
rp.DEBUG_PROFILE()
```

> 如使用配置文件,则配置文件中的 module_name 为项目名称

配置文件格式:

```
{
    "name": "debug_profile",
    "modules": [
        {
            "module_name": "test",
            "profile_service_port": 6000,
            "profile_output_dir": "../ppdemo/pdata",
            "profile_seconds": 60
        },
        {
            "module_name": "ppdemo",
            "profile_service_port": 6100,
            "profile_output_dir": "../ppdemo/pdata/debug",
            "profile_seconds": 65
        },
        {
            "module_name": "domac",
            "profile_service_port": 6200,
            "profile_output_dir": "../ppdemo/pdata",
            "profile_seconds": 60
        }
    ]
}
```

### 快照文件检测例子：

程序运行过程中，调用 rp 的开发api

```
curl http://localhost:10029/rp?mode=1,4
```

上述的调用会异步生成多个快照文件,若只想生成单个性能快照,可以参考如下:

```
生成CPU快照
curl http://localhost:10029/rp?mode=1

生成MOMERY快照
curl http://localhost:10029/rp?mode=2

生成BLOCK快照
curl http://localhost:10029/rp?mode=3

生成TRACE快照
curl http://localhost:10029/rp?mode=4
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
