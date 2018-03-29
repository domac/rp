package rp

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

type profileMux struct {
	cpuProfile string
	memProfile string
	port       uint32
	access     bool
}

func (p *profileMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/rp" {
		if !p.access {
			return
		}
		log.Println("get a command to create profile")
		var wg sync.WaitGroup
		wg.Add(2)
		p.access = false
		go ProfileCPU(p.cpuProfile, &wg)
		go ProfileMEM(p.memProfile, &wg)
		wg.Wait()
		p.access = true
		return
	}
	http.NotFound(w, r)
	return
}

func StartProfile(port int, cpuprofile, memprofile string) error {
	mux := &profileMux{
		cpuProfile: cpuprofile,
		memProfile: memprofile,
		port:       uint32(port),
		access:     true,
	}
	ps := fmt.Sprintf(":%d", mux.port)
	if err := http.ListenAndServe(ps, mux); err != nil {
		log.Fatalf("Profile  Server Failed: %v", err)
	}

	return nil
}

func ProfileCPU(cpuprofile string, wg *sync.WaitGroup) {
	if cpuprofile != "" {
		//检测cpu profile 配置
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}

		pprof.StartCPUProfile(f)
		time.AfterFunc(30*time.Second, func() {
			pprof.StopCPUProfile()
			f.Close()
			log.Println("Stop cpu profiling after 30 seconds")
			wg.Done()
		})
	} else {
		wg.Done()
	}
}

func ProfileMEM(memprofile string, wg *sync.WaitGroup) {

	if memprofile != "" {
		//检测memory profile 配置
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal(err)
		}
		time.AfterFunc(30*time.Second, func() {
			pprof.WriteHeapProfile(f)
			f.Close()
			log.Println("Stop memory profiling after 30 seconds")
			wg.Done()
		})
	} else {
		wg.Done()
	}
}
