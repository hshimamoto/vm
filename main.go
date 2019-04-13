// vm
//
// MIT License Copyright(c) 2018 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "bytes"
    "io/ioutil"
    "os"
    "fmt"
    "strings"

    "github.com/hshimamoto/vm/qemu"
    // for list
    "github.com/mitchellh/go-ps"
)

func launch(opts []string) {
    vm, err := qemu.FromConfig("config", opts)
    if err != nil {
	return
    }
    cmd := vm.Qemu()

    err = cmd.Run()
    // daemonize and return
    if err != nil {
	fmt.Printf("Run %v\n", err)
    }
}

func list(opts []string) {
    processes, err := ps.Processes()
    if err != nil {
	return
    }
    for _, p := range processes {
	pid := p.Pid()
	bin := p.Executable()
	if strings.Index(bin, "qemu") == -1 {
	    continue
	}
	//fmt.Printf("%d: %s\n", pid, bin)
	cmdline := fmt.Sprintf("/proc/%d/cmdline", pid)
	f, err := os. Open(cmdline)
	if err != nil {
	    // disappeared?
	    continue
	}
	data, err := ioutil.ReadAll(f)
	f.Close()
	name := ""
	disp := ""
	args := bytes.Split(data, []byte("\x00"))
	for i, arg := range args {
	    arg := string(arg)
	    if arg == "-name" {
		name = string(args[i+1])
	    } else if arg == "-display" {
		disp = string(args[i+1])
	    }
	}
	fmt.Printf("%d %s %s\n", pid, name, disp)
    }
}

func main() {
    if len(os.Args) == 1 {
	os.Exit(1)
    }
    subcmd := os.Args[1]
    fmt.Println(subcmd)
    if subcmd == "launch" {
	launch(os.Args[2:])
    } else if subcmd == "list" {
	list(os.Args[2:])
    }
}
