package main

import (
	//"crypto/sha512"
	"crypto/sha1"
	"crypto/sha512"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sync"

	blake2b "github.com/minio/blake2b-simd"
)

//U can use sha512 if u prefer
// var files = make(map[[sha512.Size]byte]string)
var files = make(map[string]string)

//var files = make(map[string]string)
var size = make(map[int64]string)
var smu, fmu sync.Mutex
var wg sync.WaitGroup
var numCPU = runtime.NumCPU()
var quitChan = make(chan bool)
var pathchan = make(chan string, numCPU)
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

func doHash(path string) [64]byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		//return nil
	}
	// fmt.Printf("SHA512 % x\n", sha512.Sum512(data))
	return sha512.Sum512(data) // get the file sha512 hash

}

func doHash2(path string) []byte {
	var zero []byte
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)

		return zero
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("SHA1 % x\n", h.Sum(nil))
	return h.Sum(nil)
}

func doHash3(path string) []byte {
	var zero []byte
	f, err := os.Open(path)
	if err != nil {
		fmt.Println(err)

		return zero
	}
	defer f.Close()
	// go get github.com/minio/blake2b-simd
	h := blake2b.New256() // or 512
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	// fmt.Printf("SHA1 % x\n", h.Sum(nil))
	return h.Sum(nil)
}

// HashAndCompare compare hash
// used as a group of worker, take input path from pathchan
// Ouput on std actions done ( duplicate found or linked )
//func HashAndCompare(path string) error {
func HashAndCompare() error {

	defer wg.Done()

	for {
		select {
		case path := <-pathchan:
			hash := string(doHash3(path))
			fmu.Lock()
			if v, ok := files[hash]; ok {
				links := hardlinkCount(path)

				if links < 2 { // dont' show  allready linked files
					fmt.Printf("%q is a duplicate of %q\n", path, v)
					//fmt.Printf("[%d links]", links-1)

					if flagInteractive {
						ret := confirm("Remove Left, Right or Skip ? [lrS] ", 3)
						switch ret {
						case 'l':
							err := os.Remove(path)
							if err != nil {
								log.Fatal(err)
							}
							fmt.Printf("removed %s\n", path)
						case 'r':
							err := os.Remove(v)
							if err != nil {
								log.Fatal(err)
							}
							fmt.Printf("removed %s\n", v)
						case 's':
							fmt.Println("Skipped")
						}
					}

					if flagLink {
						err := os.Remove(path)
						if err != nil {
							log.Fatal(err)
						}
						fmt.Printf("Removed %q,", path)

						err = os.Link(v, path)
						if err != nil {
							log.Fatal(err)
						}
						fmt.Printf("linked to %q.\n", v)

					}
				}

			} else {
				files[hash] = path // store in map for comparison
			}
			fmu.Unlock()

		case <-quitChan: // Reading on closed channel exit
			return nil
		}
	}
}

func main() {

	var flagPath string

	flag.StringVar(&flagPath, "path", "/tmp", "path to dedup")
	flag.BoolVar(&flagLink, "link", false, "rm and link")
	flag.BoolVar(&flagInteractive, "it", false, "interactive deletion")
	flag.Int64Var(&flagMinSize, "minsize", 1048576, "minimal file size")
	flag.Int64Var(&flagMaxSize, "maxsize", 67108864, "maximal file size")
	var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if *memprofile != "" {
		fmt.Println("Creating memprofile", *memprofile)
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Creating cpuprofile", *cpuprofile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	for num := 0; num < numCPU; num++ {
		wg.Add(1)
		go HashAndCompare()
	}

	err := filepath.Walk(flagPath, checkDuplicate)
	if err != nil {
		fmt.Println(err)
	}

	close(quitChan)
	wg.Wait()
}
