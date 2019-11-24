// vm/proc
//
// MIT License Copyright(c) 2019 Hiroshi Shimamoto
// vim:set sw=4 sts=4:
//
package proc

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os"
    "strings"

    "github.com/mitchellh/go-ps"
)

func Procread(pid int, file string) []string {
    contents := []string{}
    procfile := fmt.Sprintf("/proc/%d/%s", pid, file)
    f, err := os.Open(procfile)
    if err != nil {
	return contents
    }
    defer f.Close()
    data, err := ioutil.ReadAll(f)
    for _, elem := range bytes.Split(data, []byte("\x00")) {
	contents = append(contents, string(elem))
    }
    return contents
}

func GetProcesses(bin string) []int {
    processes, err := ps.Processes()
    if err != nil {
	return []int{}
    }
    procs := []int{}
    for _, p := range processes {
	if strings.Index(p.Executable(), bin) != -1 {
	    procs = append(procs, p.Pid())
	}
    }
    return procs
}
