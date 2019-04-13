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
	environ := fmt.Sprintf("/proc/%d/environ", pid)
	f2, err := os. Open(environ)
	if err != nil {
	    // disappeared?
	    continue
	}
	data2, err := ioutil.ReadAll(f2)
	f2.Close()
	vm_id := "-"
	vm_name := "-"
	vm_local_net := "-"
	envs := bytes.Split(data2, []byte("\x00"))
	for _, env := range envs {
	    env := string(env)
	    kv := strings.SplitN(env, "=", 2)
	    if len(kv) < 2 {
		continue
	    }
	    k := kv[0]
	    v := kv[1]
	    switch k {
	    case "VM_ID": vm_id = v
	    case "VM_NAME": vm_name = v
	    case "VM_LOCAL_NET": vm_local_net = v
	    }
	}
	fmt.Printf("%d %s %s %s %s %s\n", pid, name, disp, vm_id, vm_name, vm_local_net)
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
