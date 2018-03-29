package rp

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"sync"
	"time"
)

type profileMux struct {
	cpuProfile  string
	memProfile  string
	port        uint32
	access      bool
	profileTime time.Duration
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
		go p.ProfileCPU(p.cpuProfile, &wg)
		go p.ProfileMEM(p.memProfile, &wg)
		wg.Wait()
		p.access = true
		return
	}
	http.NotFound(w, r)
	return
}

func StartProfile(port int, cpuprofile, memprofile string, profileTime time.Duration) error {

	wd, _ := os.Getwd()

	if cpuprofile == "" {
		cpuprofile = path.Join(wd, "debug_profile.cpu")
	}

	if memprofile == "" {
		memprofile = path.Join(wd, "debug_profile.mem")
	}

	mux := &profileMux{
		cpuProfile:  cpuprofile,
		memProfile:  memprofile,
		port:        uint32(port),
		access:      true,
		profileTime: profileTime,
	}
	ps := fmt.Sprintf(":%d", mux.port)
	if err := http.ListenAndServe(ps, mux); err != nil {
		log.Fatalf("Profile  Server Failed: %v", err)
	}

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
