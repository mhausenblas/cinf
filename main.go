package main

import (
	"flag"
	"fmt"
	"github.com/shirou/gopsutil/mem"
	"log"
	"os"
)

const (
	BANNER = `/// cinf
	          ////////////////////////////
`
	VERSION = "0.1.0"
)

var (
	lnamespaces bool
	lcgroups    bool
)

func about() {
	log.Print(BANNER)
	log.Printf("This is cinf in version %s\n", VERSION)
}

func init() {
	flag.BoolVar(&lnamespaces, "namespaces", false, "List namespaces-related information")
	flag.BoolVar(&lcgroups, "cgroups", false, "List cgroups-related information")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, "Usage: cinf [args]\n")
		fmt.Fprint(os.Stderr, "\nPer default lists information on both namespaces and cgroups.")
		fmt.Fprint(os.Stderr, "\nYou can restrict output with following arguments:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
}

func display() {
	v, _ := mem.VirtualMemory()
	log.Printf("Total: %v, Free:%v, UsedPercent:%f%%\n", v.Total, v.Free, v.UsedPercent)
}

func main() {
	display()
}
