// vm/qemu / usb.go
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package qemu

import (
    "strings"
)

type usbhost struct {
    id string
}

type usb struct {
    bus string // attached host
    device string // device type
    drive string // drive id
}

func (u *usb)value() string {
    v := []string{}
    if u.device == "storage" {
	v = append(v, "usb-storage")
	if u.bus != "" {
	    v = append(v, "bus=" + u.bus + ".0")
	}
	v = push(v, "drive", u.drive)
    }
    return strings.Join(v, ",")
}
