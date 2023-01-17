package main

// Gofindup : simple go file dedup tool.

// History :
// 0.2 - 2019/07/03 - adding exclude pattern, and multipath
// 0.31 - xxhash or blake2b with -k choice
// 0.42 - output to a file, reusing this file as input
//

import (
	"bufio"
	"flag"
	"fmt"
	"hash"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	// go get github.com/minio/blake2b-simd ( cryptographic Hash)
	// go get "github.com/cespare/xxhash" ( faster non cryptographic Hash)
	xxhash "github.com/cespare/xxhash"
	blake2b "github.com/minio/blake2b-simd"
)

var (
	files = make(map[string]string) // Contains map[string(hash(path))]:path

	fmu, delmu, toFileM, itM sync.Mutex

	numCPU   = runtime.NumCPU()
	pathchan = make(chan string, 1024)

	flagLink, flagInteractive, flagkryptohash bool
	flagSilent, flagForceLink                 bool
	flagMinSize, flagMaxSize                  int64
	flagRmRegexp, flagIgnoreRegexp            string
	fromFile, toFile                          string
	compflagRmRegexp, compflagIgnoreRegexp    *regexp.Regexp
	toFileW                                   *bufio.Writer
)

func main() {

	var flagPath string
	var wg sync.WaitGroup

	flag.StringVar(&flagPath, "path", "/tmp,/dev/null", "path to dedup")
	flag.BoolVar(&flagLink, "link", false, "rm and link")
	flag.BoolVar(&flagSilent, "S", false, "Silent (no output)")
	flag.BoolVar(&flagForceLink, "f", false, "force relink (even with already linked files)")
	flag.BoolVar(&flagInteractive, "it", false, "interactive deletion")
	flag.BoolVar(&flagkryptohash, "k", false, "use kryptographic hash ( blake2 instead of xxhash )")
	flag.StringVar(&flagRmRegexp, "rm", "%d", "rm regexp")
	flag.StringVar(&flagIgnoreRegexp, "ignore", "", "ignore file path regexp")
	flag.StringVar(&fromFile, "fromFile", "", "compare items list from this file")
	flag.StringVar(&toFile, "toFile", "", "output duplicates files in this file")
	flag.Int64Var(&flagMinSize, "minsize", 1024*4, "minimal file size")
	flag.Int64Var(&flagMaxSize, "maxsize", 650, "maximal file size(Mo)")
	// var memprofile = flag.String("memprofile", "", "write memory profile to this file")
	// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if flagRmRegexp != "" {
		compflagRmRegexp = regexp.MustCompile(flagRmRegexp)
	}

	if flagIgnoreRegexp != "" {
		compflagIgnoreRegexp = regexp.MustCompile(flagIgnoreRegexp)
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

	for ; numCPU-1 > 0; numCPU-- {
		wg.Add(1)
		go HashAndCompare(&wg)
	}

	if len(toFile) > 0 {
		f, err := os.Create(toFile)
		if err != nil {
			log.Fatal(err)
		}
		toFileW = bufio.NewWriter(f)
	}
	if len(fromFile) > 0 {
		loadFile(fromFile)
	} else {
		for _, s := range strings.Split(flagPath, ",") {
			err := filepath.Walk(s, checkDuplicate)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
	close(pathchan)

	wg.Wait()
	if len(toFile) > 0 {
		toFileW.Flush()
	}
}

func loadFile(pfile string) {
	file, err := os.Open(pfile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		this := scanner.Text()
		info, err := os.Lstat(this)

		if err != nil || !info.IsDir() {
			checkDuplicate(this, info, err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}

func checkDuplicate(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	if !info.Mode().IsRegular() || (info.Size() < flagMinSize) || (info.Size() > flagMaxSize*1014*1024) {
		// skip dir or files ![min/Maxsize]
		return nil
	}
	if (len(flagIgnoreRegexp) == 0) || !compflagIgnoreRegexp.MatchString(path) {
		pathchan <- path
	}
	return nil
}

// doHash3 return cryptographic secure or fast hash of file.
func doHash3(path string) string {
	if path == "" {
		return ""
	}

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		f.Close()
		return ""
	}

	var h hash.Hash

	if flagkryptohash {
		h = blake2b.New256() // or 512
	} else {
		h = xxhash.New()
	}

	if _, err = io.Copy(h, f); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	if err = f.Close(); err != nil {
		log.Print(err)
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
		fmt.Printf("--- removed %s\n", f)
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
// Ouput on std actions done ( duplicate, found or linked )
func HashAndCompare(wg *sync.WaitGroup) error {

	defer wg.Done()

	for path := range pathchan {

		hash := doHash3(path)
		if hash == "" {
			continue
		}

		fmu.Lock() // Prevent files[hash] alteration ?
		if v, ok := files[hash]; ok {

			if len(toFile) > 0 { // Outputing to File
				toFileM.Lock()
				bufStr := path + "\n" + v + "\n"
				toFileW.WriteString(bufStr)
				toFileM.Unlock()

				fmu.Unlock()
				continue
			}

			fmu.Unlock() // Unlock as soon as possible
			links := hardlinkCount(path)
			if links < 2 || flagForceLink { // dont' show  allready linked files
				itM.Lock()
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
				if flagRmRegexp != "" {
					if compflagRmRegexp.MatchString(v) {
						removefile(v)
					} else if compflagRmRegexp.MatchString(path) {
						removefile(path)
					}
				}
				itM.Unlock()
			}
		} else {
			files[hash] = path // Store in map for comparison
			fmu.Unlock()       // Unlock as soon as possible
		}
	}
	return nil
}
