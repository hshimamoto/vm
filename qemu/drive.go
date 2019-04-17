// vm/qemu / drive.go
//
// MIT License Copyright(c) 2018,2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package qemu

import (
    "strings"
)

type drive struct {
    path string
    format string
    intf string
    // bus, unit, index string
    media string
}

func (d *drive)value() string {
    v := []string{}
    v = push(v, "file", d.path)
    v = push(v, "format", d.format)
    v = push(v, "if", d.intf)
    v = push(v, "media", d.media)
    return strings.Join(v, ",")
}
