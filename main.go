// vm
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "io/ioutil"
    "os"
    "fmt"
    "strings"
    "syscall"

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

    out, err := cmd.CombinedOutput()
    fmt.Printf("%s\n", string(out))
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
    fmt.Println("----------------------------------------")
    for _, vm := range proc.GetVMs() {
	fmt.Printf("%d %s %s %s %s %s %s\n",
		vm.Pid,
		vm.Name, vm.Disp,
		vm.VM_id, vm.VM_name, vm.VM_dir, vm.VM_local_net)
    }
    fmt.Println("")
    // nsnw
    fmt.Println("nsnw")
    fmt.Println("----------------------------------------")
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
	    u := ""
	    if data, err := ioutil.ReadFile("config"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
		    if line == "" || line[0] == '#' {
			continue
		    }
		    f := strings.Fields(line)
		    if f[0] == "user" {
			u = f[2]
			break
		    }
		}
	    }
	    args := []string{"ssh", "-p", "10022", "-i", "id_ecdsa"}
	    if u != "" {
		args = append(args, "-l", u)
	    }
	    args = append(args, vm.VM_local_net)
	    err := syscall.Exec("/usr/bin/ssh", args, os.Environ())
	    fmt.Printf("Exec: %v\n", err)
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
