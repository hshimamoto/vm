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

func procread(pid int, file string) []string {
    contents := []string{}
    procfile := fmt.Sprintf("/proc/%d/%s", pid, file)
    f, err := os.Open(procfile)
    if err != nil {
	return contents
    }
    defer f.Close()
    data, err := ioutil.ReadAll(f)
    for _, elem := range bytes.Split(data, []byte("\x00")) {
	contents = append(contents, string(elem))
    }
    return contents
}

func launch(opts []string) {
    cwd, _ := os.Getwd()
    vm, err := qemu.FromConfig(cwd, "config", opts)
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
	args := procread(pid, "cmdline")
	if len(args) == 0 {
	    continue
	}
	name := ""
	disp := ""
	for i, arg := range args {
	    switch arg {
	    case "-name": name = args[i + 1]
	    case "-disp": disp = args[i + 1]
	    }
	}
	envs := procread(pid, "environ")
	if len(envs) == 0 {
	    continue
	}
	vm_id := "-"
	vm_name := "-"
	vm_dir := "-"
	vm_local_net := "-"
	for _, env := range envs {
	    kv := strings.SplitN(env, "=", 2)
	    if len(kv) < 2 {
		continue
	    }
	    k := kv[0]
	    v := kv[1]
	    switch k {
	    case "VM_ID": vm_id = v
	    case "VM_NAME": vm_name = v
	    case "VM_DIR": vm_dir = v
	    case "VM_LOCAL_NET": vm_local_net = v
	    }
	}
	fmt.Printf("%d %s %s %s %s %s %s\n",
		pid, name, disp, vm_id, vm_name, vm_dir, vm_local_net)
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
