package main

import (
	"crypto/sha1"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

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
