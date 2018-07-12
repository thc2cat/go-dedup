package main

// Gofindup : simple go file dedup tool.

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	// go get github.com/minio/blake2b-simd
	blake2b "github.com/minio/blake2b-simd"
)

var (
	files = make(map[string]string)

	smu, fmu, delmu                                      sync.Mutex
	wg                                                   sync.WaitGroup
	numCPU                                               = runtime.NumCPU() + 2
	pathchan                                             = make(chan string, 512)
	flagLink, flagInteractive, flagSilent, flagForceLink bool
	flagMinSize, flagMaxSize                             int64
	flagRegexp                                           string
	compflagRegexp                                       *regexp.Regexp
)

func checkDuplicate(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	if !info.Mode().IsRegular() || (info.Size() < flagMinSize) || (info.Size() > flagMaxSize) {
		// skip dir or files ![min/Maxsize]
		return nil
	}
	// fmt.Println(path)
	pathchan <- path
	return nil
}

// doHash3 return cryptographic secure hash of path file.
func doHash3(path string) string {
	if path == "" {
		return ""
	}

	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return ""
	}

	// go get github.com/minio/blake2b-simd
	h := blake2b.New256() // or 512
	if _, err := io.Copy(h, f); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return string(h.Sum(nil))
}

func removefile(f string) {
	delmu.Lock()
	defer delmu.Unlock()
	err := os.Remove(f)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if !flagSilent {
		fmt.Printf("removed %s\n", f)
	}
}

func removeandlinkfile(path, v string) {
	delmu.Lock()
	defer delmu.Unlock()
	err := os.Remove(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if !flagSilent {
		fmt.Printf("« Removed,")
	}
	err = os.Link(v, path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if !flagSilent {
		fmt.Printf(" and linked. »\n")
	}
}

// HashAndCompare compare hash
// used as a group of worker, take input path from pathchan
// Ouput on std actions done ( duplicate found or linked )
func HashAndCompare() error {

	defer wg.Done()

	for path := range pathchan {

		hash := doHash3(path)
		fmu.Lock()
		if v, ok := files[hash]; ok {
			fmu.Unlock() // Unlock as soon as possible
			links := hardlinkCount(path)
			if links < 2 || flagForceLink { // dont' show  allready linked files
				if !flagSilent {
					fmt.Printf("┌ %q\n└ %q\n", path, v)
				}
				if flagInteractive && !flagSilent {
					fmu.Lock() // Lock for interactive delete
					ret := confirm("Remove line 1, 2 or Skip ? [12S] ", 3)
					switch ret {
					case '1':
						removefile(path)
					case '2':
						removefile(v)
					case 's':
						fmt.Println("Skipped")
					}
					fmu.Unlock()
				}
				if flagLink {
					removeandlinkfile(path, v)
				}
				if flagRegexp != "" {
					if compflagRegexp.MatchString(v) {
						removefile(v)
					} else if compflagRegexp.MatchString(path) {
						removefile(path)
					}
				}
			}
		} else {
			files[hash] = path // Store in map for comparison
			fmu.Unlock()       // Unlock as soon as possible
		}
	}
	return nil
}

func main() {

	var flagPath string

	flag.StringVar(&flagPath, "path", "/tmp", "path to dedup")
	flag.BoolVar(&flagLink, "link", false, "rm and link")
	flag.BoolVar(&flagSilent, "S", false, "Silent (no output)")
	flag.BoolVar(&flagForceLink, "f", false, "force relink (even with already linked files")
	flag.BoolVar(&flagInteractive, "it", false, "interactive deletion")
	flag.Int64Var(&flagMinSize, "minsize", 1024*4, "minimal file size")
	flag.Int64Var(&flagMaxSize, "maxsize", 650*1014*1024, "maximal file size")
	flag.StringVar(&flagRegexp, "regexp", "%d", "regexp for deletion of duplicate")
	// var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if flagRegexp != "" {
		compflagRegexp = regexp.MustCompile(flagRegexp)
	}
	// if *memprofile != "" {
	// 	fmt.Println("Creating memprofile", *memprofile)
	// 	f, err := os.Create(*memprofile)
	// 	if err != nil {
	// 		log.Fatal("could not create memory profile: ", err)
	// 	}
	// 	runtime.GC() // get up-to-date statistics
	// 	if err := pprof.WriteHeapProfile(f); err != nil {
	// 		log.Fatal("could not write memory profile: ", err)
	// 	}
	// 	f.Close()
	// }

	// if *cpuprofile != "" {
	// 	f, err := os.Create(*cpuprofile)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println("Creating cpuprofile", *cpuprofile)
	// 	pprof.StartCPUProfile(f)
	// 	defer pprof.StopCPUProfile()
	// }

	for ; numCPU > 0; numCPU-- {
		wg.Add(1)
		go HashAndCompare()
	}

	err := filepath.Walk(flagPath, checkDuplicate)
	if err != nil {
		fmt.Println(err)
	}
	close(pathchan)
	wg.Wait()
}
