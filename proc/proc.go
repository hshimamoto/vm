// vm/proc
//
// MIT License Copyright(c) 2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package proc

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"
    "strings"

    "github.com/mitchellh/go-ps"
)

func Procread(pid int, file string) []string {
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

func GetProcesses(bin string) []int {
    processes, err := ps.Processes()
    if err != nil {
	return []int{}
    }
    procs := []int{}
    for _, p := range processes {
	if strings.Index(p.Executable(), bin) != -1 {
	    procs = append(procs, p.Pid())
	}
    }
    return procs
}

type VM struct {
    Pid int
    Name string
    Disp string
    VM_id, VM_name, VM_dir, VM_local_net string
}

func GetVMs() []VM {
    vms := []VM{}
    for _, pid := range GetProcesses("qemu") {
	args := Procread(pid, "cmdline")
	if len(args) == 0 {
	    continue
	}
	envs := Procread(pid, "environ")
	if len(envs) == 0 {
	    continue
	}
	vm := VM{}
	vm.Pid = pid
	for i, arg := range args {
	    switch arg {
	    case "-name": vm.Name = args[i + 1]
	    case "-display": vm.Disp = args[i + 1]
	    }
	}
	for _, env := range envs {
	    kv := strings.SplitN(env, "=", 2)
	    switch kv[0] {
	    case "VM_ID": vm.VM_id = kv[1]
	    case "VM_NAME": vm.VM_name = kv[1]
	    case "VM_DIR": vm.VM_dir = kv[1]
	    case "VM_LOCAL_NET": vm.VM_local_net = kv[1]
	    }
	}
	vms = append(vms, vm)
    }
    return vms
}

type NSNW struct {
    Pid int
    Name string
}

func GetNSNWs() []NSNW {
    nsnws := []NSNW{}
    for _, pid := range GetProcesses("nsnw") {
	envs := Procread(pid, "environ")
	if len(envs) == 0 {
	    continue
	}
	nsnw := NSNW{}
	nsnw.Pid = pid
	for _, env := range envs {
	    kv := strings.SplitN(env, "=", 2)
	    if len(kv) < 2 {
		continue
	    }
	    switch kv[0] {
	    case "NSNW_NAME": nsnw.Name = kv[1]
	    }
	}
	nsnws = append(nsnws, nsnw)
    }
    return nsnws
}

func GetNSNW(name string) *NSNW {
    for _, nsnw := range GetNSNWs() {
	if nsnw.Name == name {
	    return &nsnw
	}
    }
    return nil
}
