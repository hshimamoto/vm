// tools/sw
//
// MIT License Copyright(c) 2018 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package main

import (
    "fmt"
    "net"
    "os"
)

type Plug struct {
    target string
    conn net.Conn
    state int
    learn map[string]int
}

var plugs map[string]*Plug

func (plug *Plug)miss(dst string) bool {
    v, ok := plug.learn[dst]
    if ok && v == 1 {
	return false
    }
    return true
}

func (plug *Plug)communication() {
    defer func() { plug.state = -1 }()
    conn := plug.conn
    var r int
    for {
	szbuf := make([]byte, 4)
	r, _ = conn.Read(szbuf)
	if r <= 0 {
	    return
	}
	sz := (szbuf[0] << 24) | (szbuf[1] << 16) | (szbuf[2] << 8) | szbuf[3]
	//fmt.Printf("packet: %d\n", sz)
	msgbuf := make([]byte, sz)
	r, _ = conn.Read(msgbuf)
	if r <= 0 {
	    return
	}
	// show 1st 12 bytes + 4
	//fmt.Println(msgbuf[:16])
	dst := msgbuf[:6]
	src := msgbuf[6:12]
	// learn src
	plug.learn[string(src)] = 1
	for k, v := range plugs {
	    if k == plug.target {
		continue
	    }
	    if (src[0] & 0x01) == 0 {
		v.learn[string(src)] = 0
	    }
	    if (dst[0] & 0x01) == 0 {
		if v.miss(string(dst)) {
		    continue
		}
	    }
	    //fmt.Printf("sent to %s\n", k)
	    v.conn.Write(szbuf)
	    v.conn.Write(msgbuf)
	}
    }
}

func connect(target string) {
    plug := &Plug{ target: target }
    plugs[target] = plug
    //
    conn, err := net.Dial("tcp", target)
    if err != nil {
	fmt.Printf("connect error: %s\n", err)
	return
    }
    plug.conn = conn
    plug.learn = map[string]int{}
    go plug.communication()
}

func disconnect(target string) {
}

func control(conn net.Conn) {
    defer conn.Close()

    buf := make([]byte, 256)
    r, _ := conn.Read(buf)
    if r <= 0 {
	return
    }

    msg := string(buf[:r])
    op := msg[0]
    target := msg[1:]
    fmt.Printf("op: %c target: %s\n", op, target)
    switch op {
    case '+': connect(target)
    case '-': disconnect(target)
    }
}

func instance(name string) {
    fmt.Println(name)
    sockpath := fmt.Sprintf("/tmp/vm-sw-%s.sock", name)
    fmt.Println(sockpath)

    // remove
    os.Remove(sockpath)

    l, err := net.Listen("unix", sockpath)
    if err != nil {
	fmt.Printf("error: %s\n", err)
	return
    }

    for {
	conn, err := l.Accept()
	if err != nil {
	    continue
	}
	control(conn)
    }
}

func main() {
    name := fmt.Sprintf("%d", os.Getpid())
    if len(os.Args) > 1 {
	name = os.Args[1]
    }
    plugs = map[string]*Plug{}
    instance(name)
}
