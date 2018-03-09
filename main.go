package main

// Gofindup : simple go file dedup tool.

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	blake2b "github.com/minio/blake2b-simd"
)

var files = make(map[string]string)

var smu, fmu sync.Mutex
var wg sync.WaitGroup
var numCPU = runtime.NumCPU() + 2
var pathchan = make(chan string, 256)
var flagLink, flagInteractive bool
var flagMinSize, flagMaxSize int64

func checkDuplicate(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err)
		return nil
	}
	if !info.Mode().IsRegular() || (info.Size() < flagMinSize) || (info.Size() > flagMaxSize) {
		// skip dir or files ![min/Maxsize]
		return nil
	}
	pathchan <- path
	return nil
}

// doHash3 return cryptographic secure hash of path file.
func doHash3(path string) string {
	if path == "" {
		return ""
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer f.Close()

	// go get github.com/minio/blake2b-simd
	h := blake2b.New256() // or 512
	if _, err := io.Copy(h, f); err != nil {
		log.Println(err)
	}
	// fmt.Printf("SHA1 % x\n", h.Sum(nil))
	return string(h.Sum(nil))
}

func removefile(f string) {
	err := os.Remove(f)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("removed %s\n", f)
}

func removeandlinkfile(path, v string) {
	err := os.Remove(path)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf("Removed %q,", path)

	err = os.Link(v, path)
	if err != nil {
		log.Println(err)
	}
	fmt.Printf(" linked to %q.\n", v)

}

// HashAndCompare compare hash
// used as a group of worker, take input path from pathchan
// Ouput on std actions done ( duplicate found or linked )
//func HashAndCompare(path string) error {
func HashAndCompare() error {

	defer wg.Done()

	for path := range pathchan {

		hash := doHash3(path)
		fmu.Lock()
		if v, ok := files[hash]; ok {
			fmu.Unlock() // Unlock as soon as possible
			links := hardlinkCount(path)
			if links < 2 { // dont' show  allready linked files
				fmt.Printf("┌ %q\n└ %q\n", path, v)
				if flagInteractive {
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
	flag.BoolVar(&flagInteractive, "it", false, "interactive deletion")
	flag.Int64Var(&flagMinSize, "minsize", 1048576, "minimal file size")
	flag.Int64Var(&flagMaxSize, "maxsize", 67108864, "maximal file size")
	// var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

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

	for num := 0; num < numCPU; num++ {
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
