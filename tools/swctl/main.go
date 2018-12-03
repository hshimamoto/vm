// tools/swctl
//
// MIT License Copyright(c) 2018 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "fmt"
    "net"
    "os"
    "time"
)

func instance(name string) {
    fmt.Println(name)
}

func main() {
    if len(os.Args) <= 3 {
	os.Exit(1)
    }
    cmd := os.Args[1]
    name := os.Args[2]
    param := os.Args[3]

    sockpath := fmt.Sprintf("/tmp/vm-sw-%s.sock", name)
    msg := ""

    if cmd == "connect" {
	msg = "+"
    } else if cmd == "disconnect" {
	msg = "-"
    } else {
	fmt.Printf("unknown command: %s\n", cmd)
	os.Exit(1)
    }

    msg += param

    // try to dial and send command
    conn, err := net.Dial("unix", sockpath)
    if err != nil {
	fmt.Printf("control error: %s\n", err)
	os.Exit(1)
    }

    conn.Write([]byte(msg))

    // wait a bit
    time.Sleep(time.Second)
}
