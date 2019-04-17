// vm/qemu / network.go
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package qemu

import (
    "fmt"
    "strings"
)

type nic struct {
    driver string
    netdev string
    mac string
}

func (n *nic)value() string {
    v := []string{ n.driver }
    v = push(v, "netdev", n.netdev)
    v = push(v, "mac", n.mac)
    return strings.Join(v, ",")
}

type network struct {
    nettype string
    netdev string
    // user
    hostfwds []string // like tcp:127.0.0.1:10080-:80
    guestfwds []string
    proxy string
    restrict string
    // socket
    localIP string
}

func (n *network)value() string {
    v := []string{ n.nettype }
    v = push(v, "id", n.netdev)
    // user
    switch n.nettype {
    case "user":
	for _, fwd := range n.hostfwds {
	    v = push(v, "hostfwd", fwd)
	}
	for _, fwd := range n.guestfwds {
	    f := strings.Replace(fwd, "+", " ", -1)
	    v = push(v, "guestfwd", f)
	}
	v = push(v, "proxy", n.proxy)
	v = push(v, "restrict", n.restrict)
    case "socket":
	v = push(v, "listen", fmt.Sprintf("%s:1111", n.localIP))
    default:
	// unknown
    }
    return strings.Join(v, ",")
}
