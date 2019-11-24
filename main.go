// vm
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "os"
    "fmt"
    "strings"

    "github.com/hshimamoto/vm/proc"
    "github.com/hshimamoto/vm/qemu"
    "github.com/hshimamoto/vm/cloudinit"
)

func cinit(opts []string) {
    cwd, _ := os.Getwd()
    err := cloudinit.Generate(cwd, "config", opts)
    if err != nil {
	return
    }
    fmt.Println("Generated")
}

func launch(opts []string) {
    cwd, _ := os.Getwd()
    vm, err := qemu.FromConfig(cwd, "config", opts)
    if err != nil {
	return
    }
    prepare := vm.Prepare()
    if prepare != nil {
	out, err := prepare.Output()
	if err != nil {
	    fmt.Printf("Prepare %v\n", err)
	}
	fmt.Println(string(out))
	return
    }
    cmd := vm.Qemu()

    err = cmd.Run()
    // daemonize and return
    if err != nil {
	fmt.Printf("Run %v\n", err)
    }
    // Post commands
    posts := vm.Post()
    for _, cmd := range posts {
	err := cmd.Run()
	if err != nil {
	    fmt.Printf("Run %v\n", err)
	}
    }
}

func list(opts []string) {
    psvm := proc.GetProcesses("qemu")
    // vm
    fmt.Println("vm")
    for _, pid := range psvm {
	args := proc.Procread(pid, "cmdline")
	if len(args) == 0 {
	    continue
	}
	name := ""
	disp := ""
	for i, arg := range args {
	    switch arg {
	    case "-name": name = args[i + 1]
	    case "-display": disp = args[i + 1]
	    }
	}
	envs := proc.Procread(pid, "environ")
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
    // nsnw
    psnsnw := proc.GetProcesses("nsnw")
    fmt.Println("nsnw")
    for _, pid := range psnsnw {
	envs := proc.Procread(pid, "environ")
	nsnw_name := "-"
	for _, env := range envs {
	    kv := strings.SplitN(env, "=", 2)
	    if len(kv) < 2 {
		continue
	    }
	    k := kv[0]
	    v := kv[1]
	    switch k {
	    case "NSNW_NAME": nsnw_name = v
	    }
	}
	fmt.Printf("%d %s\n", pid, nsnw_name)
    }
}

func main() {
    if len(os.Args) == 1 {
	os.Exit(1)
    }
    subcmd := os.Args[1]
    fmt.Println(subcmd)
    switch subcmd {
    case "cloudinit":
	cinit(os.Args[2:])
    case "launch":
	launch(os.Args[2:])
    case "list":
	list(os.Args[2:])
    case "help":
	fmt.Println("vm <cloudinit|launch|list>");
    }
}
