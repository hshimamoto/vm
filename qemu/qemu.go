// vm/qemu
//
// MIT License Copyright(c) 2018,2019,2020,2021 Hiroshi Shimamoto
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

    "vm/proc"
)

func push(a []string, k, v string) []string {
    if v == "" {
	return a
    }
    return append(a, k + "=" + v)
}

func keyval(s string) (string, string) {
    kv := strings.SplitN(s, "=", 2)
    if len(kv) != 2 {
	return strings.TrimSpace(kv[0]), ""
    }
    return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
}

type virtfs struct {
    driver string
    id string
    path string
    mount_tag string
    security_model string
    readonly bool
}

func (f *virtfs)value() string {
    v := []string{ f.driver }
    v = push(v, "id", f.id)
    v = push(v, "path", f.path)
    v = push(v, "mount_tag", f.mount_tag)
    v = push(v, "security_model", f.security_model)
    if f.readonly {
	v = append(v, "readonly")
    }
    return strings.Join(v, ",")
}

type ovmf struct {
    code, vars string
}

type VMConfig struct {
    dir string
    name string
    id int
    cpu, smp, mem string
    defaults bool
    localtime bool
    drives []drive
    hd0 drive
    nics []nic
    networks []network
    sound string
    tablet string
    vga string
    // usb
    usbhosts []usbhost
    usbdevs []usb
    //display string
    noreboot bool
    bootmenu string
    virtfs []virtfs
    ovmf ovmf
    //
    qemuexec string
    //
    nsnw *nsnw
    //
    opts map[string]string
    //
    args []string
}

func (vm *VMConfig)Prepare() *exec.Cmd {
    // check tap
    for _, net := range vm.networks {
	if net.nsnwpid == "" {
	    continue
	}
	if net.nsnwtapfd != "" {
	    continue
	}
	args := []string{net.nsnwpid, net.nsnwtap}
	args = append(args, os.Args...)
	fmt.Printf("prepare nstap %v\n", args)
	return exec.Command("nstap", args...)
    }
    return nil
}

func (vm *VMConfig)Post() []*exec.Cmd {
    cmds := []*exec.Cmd{}
    add_nsexec := func(args ...string) []*exec.Cmd {
	return append(cmds, exec.Command("nsexec", args...))
    }
    for _, net := range vm.networks {
	if net.nsnwpid == "" {
	    continue
	}
	// ip link add <bridge> type bridge
	cmds = add_nsexec(net.nsnwpid, "ip", "link", "add", net.nsnwbr, "type", "bridge")
	// ip link set <bridge> up
	cmds = add_nsexec(net.nsnwpid, "ip", "link", "set", net.nsnwbr, "up")
	// ip link set <tapname> master <bridge>
	cmds = add_nsexec(net.nsnwpid, "ip", "link", "set", net.nsnwtap, "master", net.nsnwbr)
	// ip link set <tapname> up
	cmds = add_nsexec(net.nsnwpid, "ip", "link", "set", net.nsnwtap, "up")
    }
    return cmds
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
    if vm.localtime {
	vm.push("-localtime")
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
    for _, usbhost := range vm.usbhosts {
	vm.push("-device", "qemu-xhci,id=" + usbhost.id)
    }
    for _, usbdev := range vm.usbdevs {
	vm.push("-device", usbdev.value())
    }
    for _, virtfs := range vm.virtfs {
	vm.push("-virtfs", virtfs.value())
    }
    if vm.ovmf.code != "" {
	vm.push("-drive", "if=pflash,format=raw,readonly,file=" + vm.ovmf.code)
    }
    if vm.ovmf.vars != "" {
	vm.push("-drive", "if=pflash,format=raw,file=" + vm.ovmf.vars)
    }
    if vm.noreboot {
	vm.push("-no-reboot")
    }
    vm.push("-serial", "null")
    vm.pushif("-soundhw", vm.sound)
    vm.pushif("-usbdevice", vm.tablet)
    vm.pushif("-vga", vm.vga)
    // display vnc=:id
    vm.push("-display", fmt.Sprintf("vnc=%s:0", vm.localIP(0)))
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
	fmt.Sprintf("VM_DIR=%s", vm.dir),
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
    // ovmf
    ovmf, ovmf_code, ovmf_vars := "", "", ""
    filepath.Walk(".",
	func(path string, info os.FileInfo, err error) error {
	    if err != nil {
		return err
	    }
	    file := info.Name()
	    // OVMF.fd?
	    switch file {
	    case "OVMF.fd": ovmf = file
	    case "OVMF_CODE.fd": ovmf_code = file
	    case "OVMF_VARS.fd": ovmf_vars = file
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
	vm.drives = append(vm.drives, hd1)
    }
    if ovmf_code != "" && ovmf_vars != "" {
	vm.ovmf.code = ovmf_code
	vm.ovmf.vars = ovmf_vars
    } else if ovmf != "" {
	vm.ovmf.vars = ovmf
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
	nsnw: newnsnw(),
	virtfs: []virtfs{},
	//
	opts: map[string]string{},
	//
	hd0: drive{},
	// usb
	usbhosts: []usbhost{},
	usbdevs: []usb{},
    }
    return vm
}

func (vm *VMConfig)addOption(opt string) {
    key, val := keyval(opt)
    if val == "" {
	return
    }
    fmt.Printf("%s = %s\n", key, val)
    if key[0] == '+' {
	vm.opts[key[1:]] += " " + val
    } else {
	vm.opts[key] = val
    }
}

func (vm *VMConfig)parseOptions() error {
    hdX := make([]string, 10)
    nicX := make([]string, 10)
    usbX := make([]string, 10)
    virtfsX := make([]string, 10)
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
	if len(key) > 3 && key[:3] == "usb" {
	    // usbX
	    n, _ := strconv.Atoi(key[3:])
	    usbX[n] = val
	    fmt.Printf("usb%d %s\n", n, val)
	    continue
	}
	if len(key) > 6 && key[:6] == "virtfs" {
	    n, _ := strconv.Atoi(key[6:])
	    virtfsX[n] = val
	    fmt.Printf("virtfs%d %s\n", n, val)
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
	case "localtime": if val != "0" { vm.localtime = true }
	case "noshut": if val != "0" { vm.noreboot = true }
	case "defaults": if val != "0" { vm.defaults = true }
	case "cdrom": vm.drives = append(vm.drives, drive{ path: val, intf: "ide", media: "cdrom"})
	case "virtfs": virtfsX[0] = val
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
	nic.mac = fmt.Sprintf("52:54:00:%02x:%02x:%02x", vm.id / 256, vm.id % 256, i)
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
		// TODO: post script
		continue
	    }
	    if param[:4] == "tap=" {
		net.nettype = "tap"
		net.ifname = param[4:]
		continue
	    }
	    if param[:5] == "nsnw=" {
		net.nettype = "tap"
		net.nsnwtap = fmt.Sprintf("tap%s%d", vm.name, i)
		// check env
		key := fmt.Sprintf("NSTAPFD_%s", net.nsnwtap)
		val, ok := os.LookupEnv(key)
		if ok {
		    net.nsnwtapfd = val
		    fmt.Printf("%s=%s\n", net.nsnwtap, net.nsnwtapfd)
		}
		net.nsnwopt = param[5:]
		opts := strings.Split(net.nsnwopt, ",")
		for _, kv := range opts {
		    key, val := keyval(kv)
		    switch key {
		    case "name": net.nsnwname = val
		    case "path": net.nsnwpath = val
		    case "br": net.nsnwbr = val
		    }
		}
		// get pid and tap
		if net.nsnwpath != "" {
		    net.nsnwpid = vm.nsnw.getpid(net.nsnwpath)
		} else {
		    if net.nsnwname == "" {
			return fmt.Errorf("nsnw: bad opt");
		    }
		    nsnw := proc.GetNSNW(net.nsnwname)
		    if nsnw == nil {
			return fmt.Errorf("nsnw: no nsnw name=%s", net.nsnwname);
		    }
		    net.nsnwpid = fmt.Sprintf("%d", nsnw.Pid)
		}
		fmt.Printf("nsnw pid=%s tapname=%s\n", net.nsnwpid, net.nsnwtap)
		continue
	    }
	    if param[:4] == "mac=" {
		mac := param[4:]
		if mac != "auto" {
		    nic.mac = mac
		}
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
    // usbX
    for i := 0; i < 10; i++ {
	if usbX[i] == "" {
	    continue
	}
	usbdev := usb{}
	// usb0 = storage=path bus=xhci
	params := strings.Split(usbX[i], " ")
	for _, param := range params {
	    if param[:8] == "storage=" {
		path := param[8:]
		a := strings.Split(path, ".")
		format := "raw"
		if a[1] == "qcow2" {
		    format = "qcow2"
		}
		id := fmt.Sprintf("usbstorage%d", i)
		// create drive
		storage := drive{
		    intf: "none",
		    id: id,
		    path: path,
		    format: format,
		}
		vm.drives = append(vm.drives, storage)
		usbdev.device = "storage"
		usbdev.drive = id
		continue
	    }
	    if param[:4] == "bus=" {
		bus := param[4:]
		ok := false
		// lookup hosts
		for _, usbhost := range vm.usbhosts {
		    if usbhost.id == bus {
			ok = true
			break
		    }
		}
		if !ok {
		    vm.usbhosts = append(vm.usbhosts, usbhost{ id: bus })
		}
		usbdev.bus = bus
		continue
	    }
	}
	vm.usbdevs = append(vm.usbdevs, usbdev)
    }
    // virtfsA
    for i := 0; i < 10; i++ {
	if virtfsX[i] == "" {
	    continue
	}
	fmt.Printf("DEBUG: virtfs %d %s\n", i, virtfsX[i])
	v := virtfs{
	    driver: "local",
	    id: fmt.Sprintf("virtfs%d", i),
	    path: virtfsX[i],
	    mount_tag: "ground",
	    security_model: "none",
	    readonly: false,
	}
	if i > 0 {
	    v.mount_tag = fmt.Sprintf("ground%d", i)
	}
	vals := strings.Split(virtfsX[i], " ")
	for _, val := range vals {
	    switch val {
	    case "readonly": v.readonly = true
	    default: v.path = val
	    }
	}
	vm.virtfs = append(vm.virtfs, v)
    }
    return nil
}

func FromConfig(dir, path string, opts []string) (*VMConfig, error) {
    f, err := os.Open(path)
    if err != nil {
	return nil, fmt.Errorf("FromConfig: %v", err)
    }
    vm := NewVM("new")
    vm.dir = dir
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
    if err := vm.parseOptions(); err != nil {
	fmt.Printf("parse error: %v\n", err)
	return nil, err
    }
    vm.localSetup()
    return vm, nil
}
