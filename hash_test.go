package main

import (
	"bytes"
	"fmt"
	"testing"
)

var file1 = "test1"

// go test -bench=.
func BenchmarkHash(b *testing.B) {

	if doHash(file1) != doHash(file1) {
		fmt.Println("Sorry doHash doesnt match")
	}
}
func BenchmarkHash2(b *testing.B) {

	if bytes.Compare(doHash2(file1), doHash2(file1)) != 0 {
		fmt.Println("Sorry doHash2 doesnt match")
	}
}

func BenchmarkHash3(b *testing.B) {

	if bytes.Compare(doHash3(file1), doHash3(file1)) != 0 {
		fmt.Println("Sorry doHash3 doesnt match")
	}
}
