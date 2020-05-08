// vm/cloudinit
//
// MIT License Copyright(c) 2019,2020 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package cloudinit

import (
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "strconv"
    "strings"

    "github.com/kdomanski/iso9660"
)

func keyval(s string) (string, string) {
    kv := strings.SplitN(s, "=", 2)
    if len(kv) != 2 {
	return strings.TrimSpace(kv[0]), ""
    }
    return strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
}

type CloudConfig struct {
    name string
    id int
    user string
    key string
    opts map[string]string
}

func (cc *CloudConfig)localIP(inst int) string {
    id8h := cc.id / 256
    id8l := cc.id % 256
    return fmt.Sprintf("127.%d.%d.%d", id8h, id8l, inst)
}

func (cc *CloudConfig)parseOptions() {
    for key, val := range cc.opts {
	switch key {
	case "name": cc.name = val
	case "id": cc.id, _ = strconv.Atoi(val)
	case "user": cc.user = val
	case "key": cc.key = val
	}
    }
    // set default
    if cc.key == "" {
	cc.key = "id_ecdsa"
    }
    if cc.user == "" {
	cc.key = "ubuntu"
    }
}

func (cc *CloudConfig)addOption(opt string) {
    key, val := keyval(opt)
    if val == "" {
	return
    }
    fmt.Printf("%s = %s\n", key, val)
    if key[0] == '+' {
	cc.opts[key[1:]] += " " + val
    } else {
	cc.opts[key] = val
    }
}

func (cc *CloudConfig)keygen() error {
    if _, err := os.Stat(cc.key); err == nil {
	// already have
	return nil
    }
    typ := "ecdsa"
    if cc.key == "id_rsa" {
	typ = "rsa"
    }
    return exec.Command("ssh-keygen", "-t", typ, "-f", cc.key, "-N", "").Run()
}

func (cc *CloudConfig)gen_userdata() error {
    f, err := os.Create("user-data")
    if err != nil {
	return fmt.Errorf("userdata: Create %v", err)
    }
    defer f.Close()
    var lines []string
    // header
    lines = append(lines, "#cloud-config")
    //
    lines = append(lines, "manage_etc_hosts: true")
    lines = append(lines, "hostname: " + cc.name)
    lines = append(lines, "timezone: Asia/Tokyo")
    // users
    lines = append(lines, "users:")
    lines = append(lines, "  - name: " + cc.user)
    lines = append(lines, "    ssh-authorized-keys:")
    pubkey, err := func() (string, error) {
	pub, err := os.Open(cc.key + ".pub")
	if err != nil {
	    return "", err
	}
	defer pub.Close()
	data, err := ioutil.ReadAll(pub)
	if err != nil {
	    return "", err
	}
	lines := strings.Split(string(data), "\n")
	if lines[0] == "" {
	    return "", fmt.Errorf("pubkey error")
	}
	return lines[0], nil
    }()
    lines = append(lines, "      - " + pubkey)
    lines = append(lines, "    sudo: ['ALL=(ALL) NOPASSWD:ALL']")
    lines = append(lines, "    groups: [adm, cdrom, sudo, dip, plugdev]")
    lines = append(lines, "    shell: /bin/bash")
    f.Write([]byte(strings.Join(lines, "\n") + "\n"))
    return nil
}

func (cc *CloudConfig)gen_metadata() error {
    f, err := os.Create("meta-data")
    if err != nil {
	return fmt.Errorf("metadata: Create %v", err)
    }
    defer f.Close()
    f.Write([]byte("instance-id: " + cc.name + "\n"))
    return nil
}

func Generate(dir, path string, opts []string) error {
    f, err := os.Open(path)
    if err != nil {
	return fmt.Errorf("Generate: %v", err)
    }
    defer f.Close()
    cc := &CloudConfig{}
    cc.opts = map[string]string{}
    // load config
    data, err := ioutil.ReadAll(f)
    lines := strings.Split(string(data), "\n")
    for _, line := range lines {
	if line == "" {
	    continue
	}
	if line[0] == '#' {
	    continue
	}
	cc.addOption(line)
    }
    // override
    for _, opt := range opts {
	cc.addOption(opt)
    }
    cc.parseOptions()
    // generate keys
    if err := cc.keygen(); err != nil {
	return err
    }
    // generate user-data and meta-data
    if err := cc.gen_userdata(); err != nil {
	return err
    }
    if err := cc.gen_metadata(); err != nil {
	return err
    }
    // create ISO9660 image
    writer, err := iso9660.NewWriter()
    if err != nil {
	return err
    }
    defer writer.Cleanup()
    meta, err := os.Open("meta-data")
    if err != nil {
	return err
    }
    defer meta.Close()
    err = writer.AddFile(meta, "meta-data")
    if err != nil {
	return err
    }
    user, err := os.Open("user-data")
    if err != nil {
	return err
    }
    defer user.Close()
    err = writer.AddFile(user, "user-data")
    if err != nil {
	return err
    }
    iso, err := os.OpenFile("user-data.img", os.O_WRONLY | os.O_TRUNC | os.O_CREATE, 0644)
    if err != nil {
	return err
    }
    defer iso.Close()
    if err := writer.WriteTo(iso, "cidata"); err != nil {
	return err
    }
    return nil
}
