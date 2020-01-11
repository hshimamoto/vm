// vm
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "os"
    "os/exec"
    "fmt"

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
    // vm
    fmt.Println("vm")
    for _, vm := range proc.GetVMs() {
	fmt.Printf("%d %s %s %s %s %s %s\n",
		vm.Pid,
		vm.Name, vm.Disp,
		vm.VM_id, vm.VM_name, vm.VM_dir, vm.VM_local_net)
    }
    // nsnw
    fmt.Println("nsnw")
    for _, nsnw := range proc.GetNSNWs() {
	fmt.Printf("%d %s\n", nsnw.Pid, nsnw.Name)
    }
}

func ssh(opts []string) {
    tgt := opts[0]
    for _, vm := range proc.GetVMs() {
	if (vm.Name == tgt) {
	    fmt.Printf("ssh to %s\n", tgt)
	    os.Chdir(vm.VM_dir)
	    cmd := exec.Command("ssh", "-p", "10022", "-i", "id_ecdsa", vm.VM_local_net)
	    // pipe
	    cmd.Stdin = os.Stdin
	    cmd.Stdout = os.Stdout
	    cmd.Stderr = os.Stderr
	    cmd.Run()
	    return
	}
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
    case "ssh":
	ssh(os.Args[2:])
    case "help":
	fmt.Println("vm <cloudinit|launch|list|ssh>");
    }
}
