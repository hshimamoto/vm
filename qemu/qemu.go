// vm/qemu
//
// MIT License Copyright(c) 2018 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package qemu

import (
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
)

func push(a []string, k, v string) []string {
    if v == "" {
	return a
    }
    return append(a, k + "=" + v)
}

type VMConfig struct {
    name string
    id int
    cpu, smp, mem string
    defaults bool
    drives []drive
    hd0 drive
    nics []nic
    networks []network
    sound string
    tablet string
    vga string
    //display string
    noreboot bool
    bootmenu string
    //
    qemuexec string
    //
    opts map[string]string
    //
    args []string
}

func (vm *VMConfig)Qemu() *exec.Cmd {
    vm.args = []string{}
    vm.push("-name", vm.name)
    vm.pushif("-cpu", vm.cpu)
    vm.pushif("-smp", vm.smp)
    vm.pushif("-m", vm.mem)
    vm.push("-boot", vm.bootmenu)
    if !vm.defaults {
	vm.push("-nodefaults")
    }
    for _, drive := range vm.drives {
	vm.push("-drive", drive.value())
    }
    for _, nic := range vm.nics {
	vm.push("-device", nic.value())
    }
    for _, net := range vm.networks {
	vm.push("-netdev", net.value())
    }
    if vm.noreboot {
	vm.push("-no-reboot")
    }
    vm.pushif("-soundhw", vm.sound)
    vm.pushif("-usbdevice", vm.tablet)
    vm.pushif("-vga", vm.vga)
    // display vnc=:id
    vm.push("-display", fmt.Sprintf("vnc=:%d", vm.id))
    // always on
    vm.push("-enable-kvm")
    vm.push("-daemonize")
    vm.push("-pidfile", "qemu.pid")
    vm.push("-monitor", "vc")
    fmt.Println(vm.args)
    // env
    env := []string{
	fmt.Sprintf("VM_ID=%d", vm.id),
	fmt.Sprintf("VM_NAME=%s", vm.name),
	fmt.Sprintf("VM_LOCAL_NET=%s", vm.localIP(0)),
    }
    fmt.Println(env)

    cmd := exec.Command(vm.qemuexec, vm.args...)
    cmd.Env = append(os.Environ(), env...)
    return cmd
}

func (vm *VMConfig)push(args ...string) {
    vm.args = append(vm.args, args...)
}

func (vm *VMConfig)pushif(opt, arg string) {
    if arg != "" {
	vm.args = append(vm.args, opt, arg)
    }
}

func (vm *VMConfig)localIP(inst int) string {
    id8h := vm.id / 256
    id8l := vm.id % 256
    return fmt.Sprintf("127.%d.%d.%d", id8h, id8l, inst)
}

func (vm *VMConfig)plug(device interface{}) {
}

func (vm *VMConfig)localSetup() {
    // hd0.qcow2 or hd0.raw
    hd0, hd1 := drive{}, drive{}
    filepath.Walk(".",
	func(path string, info os.FileInfo, err error) error {
	    if err != nil {
		return err
	    }
	    a := strings.Split(info.Name(), ".")
	    if len(a) != 2 {
		return nil
	    }
	    name := a[0]
	    ext := a[1]
	    if ext != "qcow2" && ext != "raw" {
		return nil
	    }
	    switch name {
	    case "hd0": hd0 = drive{ path: info.Name(), intf: "virtio", format: ext }
	    case "hd1": hd1 = drive{ path: info.Name(), intf: "virtio", format: ext }
	    }
	    return nil
	});
    if vm.hd0.path != "" {
	vm.drives = append(vm.drives, vm.hd0)
    } else {
	if hd0.path != "" {
	    vm.drives = append(vm.drives, hd0)
	}
    }
    if hd1.path != "" {
	vm.drives = append(vm.drives, hd0)
    }
}

func NewVM(name string) *VMConfig {
    vm := &VMConfig{
	name: name,
	drives: []drive{},
	nics: []nic{},
	networks: []network{},
	// default settings
	bootmenu: "menu=on,splash-time=5000",
	vga: "std",
	qemuexec: "qemu-system-x86_64",
	//
	opts: map[string]string{},
	//
	hd0: drive{},
    }
    return vm
}

func (vm *VMConfig)addOption(opt string) {
    kv := strings.SplitN(opt, "=", 2)
    if len(kv) != 2 {
	return
    }
    key := strings.TrimSpace(kv[0])
    val := strings.TrimSpace(kv[1])
    fmt.Printf("%s = %s\n", key, val)
    if key[0] == '+' {
	vm.opts[key[1:]] += " " + val
    } else {
	vm.opts[key] = val
    }
}

func (vm *VMConfig)parseOptions() {
    hdX := make([]string, 10)
    nicX := make([]string, 10)
    for key, val := range vm.opts {
	if len(key) > 2 && key[:2] == "hd" {
	    // hdX
	    n, _ := strconv.Atoi(key[2:])
	    hdX[n] = val
	    continue
	}
	if len(key) > 3 && key[:3] == "nic" {
	    // nicX
	    n, _ := strconv.Atoi(key[3:])
	    nicX[n] = val
	    fmt.Printf("nic%d %s\n", n, val)
	    continue
	}
	switch key {
	case "name": vm.name = val
	case "id": vm.id, _ = strconv.Atoi(val)
	case "cpu": vm.cpu = val
	case "smp": vm.smp = val
	case "mem": vm.mem = val
	case "vga": vm.vga = val
	case "sound": vm.sound = val
	case "qemu": vm.qemuexec = val
	case "noshut": if val != "0" { vm.noreboot = true }
	case "defaults": if val != "0" { vm.defaults = true }
	case "cdrom": vm.drives = append(vm.drives, drive{ path: val, intf: "ide", media: "cdrom"})
	}
    }
    // TODO hdX
    // check only hd0
    if hdX[0] != "" {
	vm.hd0 = drive{}
	params := strings.Split(hdX[0], " ")
	for _, param := range params {
	    if param[:3] == "if=" {
		vm.hd0.intf = param[3:]
		continue
	    }
	    if param[:5] == "path=" {
		vm.hd0.path = param[5:]
		continue
	    }
	    if param[:7] == "format=" {
		vm.hd0.format = param[7:]
		continue
	    }
	}
    }
    // nicX
    vm.nics = []nic{}
    vm.networks = []network{}
    for i := 0; i < 10; i++ {
	if nicX[i] == "" {
	    continue
	}
	netdev := fmt.Sprintf("vnic%d", i)
	nic := nic{ driver: "virtio-net", netdev: netdev }
	net := network{ nettype: "user", netdev: netdev }
	params := strings.Split(nicX[i], " ")
	for _, param := range params {
	    if param == "default" {
		nic.driver = "virtio-net"
		net.nettype = "user"
		lo := vm.localIP(i)
		net.hostfwds = []string{
		    fmt.Sprintf("tcp:%s:10022-:22", lo),
		    fmt.Sprintf("tcp:%s:10080-:80", lo),
		    fmt.Sprintf("tcp:%s:13389-:3389", lo),
		}
		continue
	    }
	    if param[:7] == "socket=" {
		net.nettype = "socket"
		net.localIP = vm.localIP(i)
		nic.mac = fmt.Sprintf("52:54:00:%02x:%02x:%02x", vm.id / 256, vm.id % 256, i)
		// TODO: post script
		continue
	    }
	    if param[:6] == "proxy=" {
		net.proxy = param[6:]
		continue
	    }
	    if param[:7] == "driver=" {
		nic.driver = param[7:]
		continue
	    }
	    if param[:8] == "hostfwd=" {
		p := strings.Replace(param[8:], "$ip", vm.localIP(i), -1)
		net.hostfwds = append(net.hostfwds, p)
		continue
	    }
	    if param[:9] == "restrict=" {
		net.restrict = param[9:]
		continue
	    }
	    if param[:9] == "guestfwd=" {
		net.guestfwds = append(net.guestfwds, param[9:])
		continue
	    }
	}
	vm.nics = append(vm.nics, nic)
	vm.networks = append(vm.networks, net)
    }
}

func FromConfig(path string, opts []string) (*VMConfig, error) {
    f, err := os.Open(path)
    if err != nil {
	return nil, fmt.Errorf("FromConfig: %v", err)
    }
    vm := NewVM("new")
    // default network option
    vm.addOption("nic0=default")
    //
    data, err := ioutil.ReadAll(f)
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
	if line == "" {
	    continue
	}
	if line[0] == '#' {
	    continue
	}
	vm.addOption(line)
    }
    // override
    for _, opt := range opts {
	vm.addOption(opt)
    }
    vm.parseOptions()
    vm.localSetup()
    return vm, nil
}
