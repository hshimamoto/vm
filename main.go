// vm
//
// MIT License Copyright(c) 2018 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "os"
    "fmt"

    "github.com/hshimamoto/vm/qemu"
)

func launch(opts []string) {
    vm, err := qemu.FromConfig("config", opts)
    if err != nil {
	return
    }
    cmd := vm.Qemu()

    fmt.Println(cmd)
}

func main() {
    if len(os.Args) == 1 {
	os.Exit(1)
    }
    subcmd := os.Args[1]
    fmt.Println(subcmd)
    if subcmd == "launch" {
	launch(os.Args[2:])
    }
}
