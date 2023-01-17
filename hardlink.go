//go:build linux || freebsd
// +build linux freebsd

package main

import (
	"errors"
	"log"
	"os"
	"syscall"
)

func hardlinkCount(filename string) int {

	// 'os.Lstat()' reads the link itself.
	// 'os.Stat()' would read the link's target.
	fi, err := os.Lstat(filename)
	if err != nil {
		log.Fatal(err)
	}

	// https://github.com/docker/docker/blob/master/pkg/archive/archive_unix.go
	// in 'func setHeaderForSpecialDevice()'
	s, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		err = errors.New("cannot convert stat value to syscall.Stat_t")
		log.Fatal(err)
	}

	// The index number of this file's inode:
	// inode := uint64(s.Ino)
	// Total number of files/hardlinks connected to this file's inode:
	return int(s.Nlink)

	// True if the file is a symlink.
	//  if fi.Mode()&os.ModeSymlink != 0 {
	// 		 link, err := os.Readlink(fi.Name())
	// 		 if err != nil {
	// 				 log.Fatal(err)
	// 		 }
	// 		 fmt.Printf("%v is a symlink to %v on inode %v.\n", filename, link, inode)
	// 		 os.Exit(0)
	//  }
	//
	//  // Otherwise, for hardlinks:
	//  fmt.Printf("The inode for %v, %v, has %v hardlinks.\n", filename, inode, nlink)
	//  if nlink > 1 {
	// 		 fmt.Printf("Inode %v has %v other hardlinks besides %v.\n", inode, nlink, filename)
	//  } else {
	// 		 fmt.Printf("%v is the only hardlink to inode %v.\n", filename, inode)
	//  }
}
