package rp

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const DEFAULT_PORT = 10029 //默认端口

type RpConfig struct {
	DebugProfile DebugProfile `toml:"debug_profile"`
}

type DebugProfile struct {
	ModulePorts     []int    `toml:"module_ports"`
	ModuleNames     []string `toml:"module_names"`
	ProfileOutpuDir string   `toml:"profile_output_dir"`
}

var g_rpconfig = new(RpConfig)

type profileMux struct {
	cpuProfile  string
	memProfile  string
	port        uint32
	started     uint32
	profileTime time.Duration
}

//debug服务
func (p *profileMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/rp" {

		if !atomic.CompareAndSwapUint32(&p.started, 0, 1) {
			log.Printf("profile already called")
			return
		}

		var wg sync.WaitGroup
		wg.Add(2)
		atomic.StoreUint32(&p.started, 1)
		go p.ProfileCPU(p.cpuProfile, &wg)
		go p.ProfileMEM(p.memProfile, &wg)
		wg.Wait()
		atomic.StoreUint32(&p.started, 0)
		w.Header().Set("Content-type", "text/html")
		io.WriteString(w, "profile finish\r\n")
		return
	}
	http.NotFound(w, r)
	return
}

//默认创建快照方式
func CreateProfile(mode int) error {
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

	if mode == 1 || mode == 3 {
		cpuprofile = path.Join(wd, "debug_profile.cpu")
	}

	if mode == 2 || mode == 3 {
		memprofile = path.Join(wd, "debug_profile.mem")
	}

	if g_rpconfig != nil {
		for idx, name := range g_rpconfig.DebugProfile.ModuleNames {
			if strings.Contains(callerFileName, name) &&
				len(g_rpconfig.DebugProfile.ModulePorts) > idx {
				port = g_rpconfig.DebugProfile.ModulePorts[idx]

				//如果是配置的模块名,则重新设置输出文件名称
				if g_rpconfig.DebugProfile.ProfileOutpuDir != "" {

					os.MkdirAll(g_rpconfig.DebugProfile.ProfileOutpuDir, 0777)

					if cpuprofile != "" {
						cpuprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir, name+"_debug_profile_cpu.prof")
					}

					if memprofile != "" {
						memprofile = path.Join(g_rpconfig.DebugProfile.ProfileOutpuDir, name+"_debug_profile_mem.prof")
					}
				}
			}
		}
	}

	return StartProfile(port, cpuprofile, memprofile, 30*time.Second)
}

func StartProfile(port int, cpuprofile, memprofile string, profileTime time.Duration) error {
	//获取调用者的标准文件名称
	mux := &profileMux{
		cpuProfile:  cpuprofile,
		memProfile:  memprofile,
		port:        uint32(port),
		profileTime: profileTime,
	}

	go func(mux *profileMux) {
		ps := fmt.Sprintf(":%d", mux.port)
		log.Printf("debug profile call : http://127.0.0.1%s/rp\n", ps)
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
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		time.AfterFunc(p.profileTime, func() {
			pprof.StopCPUProfile()
			f.Close()
			log.Println("Stop cpu profiling after 30 seconds")
			wg.Done()
		})
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
			log.Fatal(err)
		}
		time.AfterFunc(p.profileTime, func() {
			pprof.WriteHeapProfile(f)
			f.Close()
			log.Println("Stop memory profiling after 30 seconds")
			wg.Done()
		})
	} else {
		wg.Done()
	}
}

func LoadDebugProfile(path string) {
	_, err := toml.DecodeFile(path, &g_rpconfig)
	if err != nil {
		log.Println(err.Error())
	}
}
