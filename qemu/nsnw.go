// vm/qemu / nsnw
//
// MIT License Copyright(c) 2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package qemu

import (
    "fmt"
    "os"
)

type nsnw struct {
    pidmap map[string]string
}

func newnsnw() *nsnw {
    nsnw := &nsnw{
	pidmap: map[string]string{},
    }
    return nsnw
}

func (nsnw *nsnw)getpid(path string) string {
    pid, ok := nsnw.pidmap[path]
    if ok {
	return pid
    }
    // read
    f, err := os.Open(path)
    if err != nil {
	fmt.Printf("unable to open pid file %s\n", path)
	return ""
    }
    defer f.Close()
    buf := make([]byte, 32)
    n, err := f.Read(buf)
    if n == 0 {
	fmt.Printf("unable to find pid in file %s\n", path)
	return ""
    }
    nsnw.pidmap[path] = string(buf[:n])
    return nsnw.pidmap[path]
}
