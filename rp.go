package rp

import (
	"bufio"
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DEFAULT_PORT = 10029 //默认端口

	MODE_DEBUG_PROFILE_ALL    = 0
	MODE_DEBUG_PROFILE_CPU    = 1
	MODE_DEBUG_PROFILE_MEMORY = 2
	MODE_DEBUG_PROFILE_BLOCK  = 3
	MODE_DEBUG_PROFILE_TRACE  = 4

	memProfileRate = 4096
)

type RpConfig struct {
	DebugProfile DebugProfile `toml:"debug_profile"`
}

type DebugProfile struct {
	ModulePorts     []int    `toml:"module_ports"`
	ModuleNames     []string `toml:"module_names"`
	ProfileOutpuDir string   `toml:"profile_output_dir"`
	ProfileSeconds  int      `toml:"profile_seconds"`
}

var g_rpconfig = new(RpConfig)

type profileMux struct {
	cpuProfile   string
	memProfile   string
	blockProfile string
	traceProfile string
	port         uint32
	started      uint32
	profileTime  time.Duration
	ctx          context.Context
}

func isInclude(list []string, element int) bool {
	for _, v := range list {
		vi, err := strconv.Atoi(v)
		if err != nil {
			continue
		}
		if element == vi {
			return true
		}
	}
	return false
}

//debug服务
func (p *profileMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	//调用模式
	modestr := r.URL.Query().Get("mode")

	modes := strings.Split(modestr, ",")

	//mode, _ := strconv.Atoi(modestr)
	log.Printf("debug mode = %s\n", modes)

	if r.URL.Path == "/rp" {
		if !atomic.CompareAndSwapUint32(&p.started, 0, 1) {
			log.Printf("profile already called")
			return
		}
		var wg sync.WaitGroup
		atomic.StoreUint32(&p.started, 1)
		if isInclude(modes, MODE_DEBUG_PROFILE_CPU) {
			wg.Add(1)
			go p.ProfileCPU(p.cpuProfile, &wg)
		}

		if isInclude(modes, MODE_DEBUG_PROFILE_MEMORY) {
			wg.Add(1)
			go p.ProfileMEM(p.memProfile, &wg)
		}

		if isInclude(modes, MODE_DEBUG_PROFILE_BLOCK) {
			wg.Add(1)
			go p.ProfileBlock(p.blockProfile, &wg)
		}

		if isInclude(modes, MODE_DEBUG_PROFILE_TRACE) {
			wg.Add(1)
			go p.ProfileTrace(p.traceProfile, &wg)
		}
		wg.Wait()
		log.Println("profile task done !")
		atomic.StoreUint32(&p.started, 0)
		w.Header().Set("Content-type", "text/html")
		io.WriteString(w, "profile finish\r\n")
		return
	}
	http.NotFound(w, r)
	return
}

//默认创建快照方式
func CreateProfile() error {
	_, callerFileName, _, _ := runtime.Caller(1)
	srcIndex := strings.LastIndex(callerFileName, "src")
	if srcIndex > 0 {
		callerFileName = callerFileName[srcIndex+1:]
	}
	log.Printf("caller file : %s\n", callerFileName)

	port := DEFAULT_PORT
	//获取当前的工作目录
	wd, _ := os.Getwd()
	cpuprofile := ""
	memprofile := ""
	blockprofile := ""
	traceprofile := ""
	profileTimeSeconds := 30 * time.Second

	cpuprofile = path.Join(wd, "debug_profile.cpu")
	memprofile = path.Join(wd, "debug_profile.mem")
	blockprofile = path.Join(wd, "debug_profile.block")
	traceprofile = path.Join(wd, "debug_profile.trace")

	if g_rpconfig != nil {
		for idx, name := range g_rpconfig.DebugProfile.ModuleNames {
			if strings.Contains(callerFileName, name) &&
				len(g_rpconfig.DebugProfile.ModulePorts) > idx {
				port = g_rpconfig.DebugProfile.ModulePorts[idx]

				//如果是配置的模块名,则重新设置输出文件名称
				if g_rpconfig.DebugProfile.ProfileOutpuDir != "" {

					os.MkdirAll(g_rpconfig.DebugProfile.ProfileOutpuDir, 0777)

					if cpuprofile != "" {
						cpuprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir,
							name+"_debug_profile_cpu.prof")
					}

					if memprofile != "" {
						memprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir,
							name+"_debug_profile_mem.prof")
					}

					if blockprofile != "" {
						blockprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir,
							name+"_debug_profile_block.prof")
					}

					if traceprofile != "" {
						traceprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir,
							name+"_debug_profile_trace.prof")
					}

					if g_rpconfig.DebugProfile.ProfileSeconds > 0 {
						log.Printf("debug profile seconds : %d\n", g_rpconfig.DebugProfile.ProfileSeconds)
						profileTimeSeconds = time.Duration(g_rpconfig.DebugProfile.ProfileSeconds) * time.Second
					}
				}
			}
		}
	}
	//开始profile
	return StartProfile(port, cpuprofile, memprofile, blockprofile, traceprofile, profileTimeSeconds)
}

func StartProfileWithContxt(port int, cpuprofile, memprofile, blockProfile, traceprofile string, ctx context.Context) error {
	mux := &profileMux{
		cpuProfile:   cpuprofile,
		memProfile:   memprofile,
		blockProfile: blockProfile,
		traceProfile: traceprofile,
		port:         uint32(port),
		ctx:          ctx,
	}

	go func(mux *profileMux) {
		ps := fmt.Sprintf(":%d", mux.port)
		log.Printf("debug profile call [contxt]: http://127.0.0.1%s/rp\n", ps)
		if err := http.ListenAndServe(ps, mux); err != nil {
			log.Fatalf("Profile  Server Failed: %v", err)
		}
	}(mux)
	return nil
}

func StartProfile(port int, cpuprofile, memprofile, blockProfile, traceprofile string, profileTime time.Duration) error {
	//获取调用者的标准文件名称
	mux := &profileMux{
		cpuProfile:   cpuprofile,
		memProfile:   memprofile,
		blockProfile: blockProfile,
		traceProfile: traceprofile,
		port:         uint32(port),
		profileTime:  profileTime,
	}

	go func(mux *profileMux) {
		ps := fmt.Sprintf(":%d", mux.port)
		log.Printf("debug profile call : http://127.0.0.1%s/rp?mode=1\n", ps)
		if err := http.ListenAndServe(ps, mux); err != nil {
			log.Fatalf("Profile  Server Failed: %v", err)
		}
	}(mux)
	return nil
}

func (p *profileMux) ProfileCPU(cpuprofile string, wg *sync.WaitGroup) {
	log.Printf("cpu profile : %s\n", cpuprofile)
	if cpuprofile != "" {
		//检测cpu profile 配置
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatalf("profile: could not create cpu profile %q: %v", cpuprofile, err)
		}
		pprof.StartCPUProfile(f)

		if p.ctx == nil {
			time.AfterFunc(p.profileTime, func() {
				pprof.StopCPUProfile()
				f.Close()
				log.Println("cpu profiling finish")
				wg.Done()
			})
		} else {
			<-p.ctx.Done()
			pprof.StopCPUProfile()
			f.Close()
			log.Println("[contxt] cpu profiling finish")
			wg.Done()
		}
	} else {
		wg.Done()
	}
}

func (p *profileMux) ProfileMEM(memprofile string, wg *sync.WaitGroup) {
	log.Printf("memory profile : %s\n", memprofile)
	if memprofile != "" {
		//检测memory profile 配置
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatalf("profile: could not create memory profile %q: %v", memprofile, err)
		}

		old := runtime.MemProfileRate
		runtime.MemProfileRate = memProfileRate

		if p.ctx == nil {
			time.AfterFunc(p.profileTime, func() {
				pprof.Lookup("heap").WriteTo(f, 0)
				f.Close()
				runtime.MemProfileRate = old
				log.Println("memory profiling finish")
				wg.Done()
			})
		} else {
			<-p.ctx.Done()
			pprof.Lookup("heap").WriteTo(f, 0)
			f.Close()
			runtime.MemProfileRate = old
			log.Println("[context] memory profiling finish")
			wg.Done()
		}
	} else {
		wg.Done()
	}
}

func (p *profileMux) ProfileBlock(blockfile string, wg *sync.WaitGroup) {
	log.Printf("block profile : %s\n", blockfile)
	if blockfile != "" {
		f, err := os.Create(blockfile)
		if err != nil {
			log.Fatalf("profile: could not create block profile %q: %v", blockfile, err)
		}
		runtime.SetBlockProfileRate(1)

		if p.ctx == nil {
			time.AfterFunc(p.profileTime, func() {
				pprof.Lookup("block").WriteTo(f, 0)
				f.Close()
				runtime.SetBlockProfileRate(0)
				log.Println("block profiling finish")
				wg.Done()
			})
		} else {
			<-p.ctx.Done()
			pprof.Lookup("block").WriteTo(f, 0)
			f.Close()
			runtime.SetBlockProfileRate(0)
			log.Println("[context] block profiling finish")
			wg.Done()
		}
	} else {
		wg.Done()
	}
}

func (p *profileMux) ProfileTrace(tracefile string, wg *sync.WaitGroup) {
	log.Printf("trace profile : %s\n", tracefile)
	if tracefile != "" {
		f, err := os.Create(tracefile)
		if err != nil {
			log.Fatalf("profile: could not create trace profile %q: %v", tracefile, err)
		}
		trace.Start(f)

		if p.ctx == nil {
			time.AfterFunc(p.profileTime, func() {
				trace.Stop()
				f.Close()
				log.Println("trace profiling finish")
				wg.Done()
			})
		} else {
			<-p.ctx.Done()
			trace.Stop()
			f.Close()
			log.Println("[context]trace profiling finish")
			wg.Done()
		}
	} else {
		wg.Done()
	}
}

//加载debug配置
func LoadDebugProfile(path string) {
	_, err := toml.DecodeFile(path, &g_rpconfig)
	if err != nil {
		log.Println(err.Error())
	}
}

func DoTrace() func() {
	traceprofile := "./trace.out"
	bw, flush := bufferedFileWriter(traceprofile)
	trace.Start(bw)
	return func() {
		flush()
		trace.Stop()
	}
}

func bufferedFileWriter(dest string) (w io.Writer, close func()) {
	f, err := os.Create(dest)
	if err != nil {
		log.Fatal(err)
	}
	bw := bufio.NewWriter(f)
	return bw, func() {
		if err := bw.Flush(); err != nil {
			log.Fatalf("error flushing %v: %v", dest, err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}
}
