package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const (
	BANNER = `     ___                    ___          ___     
    /\__\                  /\  \        /\__\    
   /:/  /       ___        \:\  \      /:/ _/_   
  /:/  /       /\__\        \:\  \    /:/ /\__\  
 /:/  /  ___  /:/__/    _____\:\  \  /:/ /:/  /  
/:/__/  /\__\/::\  \   /::::::::\__\/:/_/:/  /   
\:\  \ /:/  /\/\:\  \__\:\~~\~~\/__/\:\/:/  /    
 \:\  /:/  /  ~~\:\/\__\\:\  \       \::/__/     
  \:\/:/  /      \::/  / \:\  \       \:\  \     
   \::/  /       /:/  /   \:\__\       \:\__\    
    \/__/        \/__/     \/__/        \/__/   
`
	VERSION = "0.1.0"
	DEBUG   = false
)

type NSTYPE uint64

const (
	NS_MOUNT  NSTYPE = iota // CLONE_NEWNS, filesystem mount points
	NS_UTS                  // CLONE_NEWUTS, nodename and NIS domain name
	NS_IPC                  // CLONE_NEWIPC, interprocess communication
	NS_PID                  // CLONE_NEWPID, process ID number space isolation
	NS_NET                  // CLONE_NEWNET, network system resources
	NS_USER                 // CLONE_NEWUSER, user and group ID number space isolation
	NS_CGROUP               // CLONE_NEWCGROUP, cgroup root directory
)

type Namespace struct {
	Type NSTYPE
	Id   string
}

var (
	lnamespaces bool
	lcgroups    bool
	lversion    bool
)

func about() {
	fmt.Printf(BANNER)
	fmt.Printf("\nThis is cinf in version %s\n", VERSION)
	fmt.Print("See also https://github.com/mhausenblas/cinf\n\n")
}

func init() {
	flag.BoolVar(&lnamespaces, "namespaces", false, "List only namespaces-related information")
	flag.BoolVar(&lcgroups, "cgroups", false, "List only cgroups-related information")
	flag.BoolVar(&lversion, "version", false, "List info about cinf including version")

	flag.Usage = func() {
		fmt.Printf("Usage: %s [args]\n\n", os.Args[0])
		fmt.Printf("Per default lists information on both namespaces and cgroups.\n")
		fmt.Println("Arguments:")
		flag.PrintDefaults()
	}
	flag.Parse()
}

func debug(m string) {
	if DEBUG {
		fmt.Printf("DEBUG: %s\n", m)
	}
}

func resolve(namespace string, processID string) (*Namespace, error) {
	nsfile := filepath.Join("/proc", processID, "ns", namespace)
	debug(nsfile)
	if content, err := os.Readlink(nsfile); err == nil {
		debug(content)
		nsnum := strings.Split(content, ":")[1]
		nsnum = nsnum[1 : len(nsnum)-1]
		ns := Namespace{}
		ns.Type = NS_NET
		ns.Id = string(nsnum)
		return &ns, nil
	} else {
		return nil, err
	}
}

func listn() {
	fmt.Println("namespaces:")

	files, _ := ioutil.ReadDir("/proc")
	for _, f := range files {
		if _, err := strconv.Atoi(f.Name()); err == nil {
			if ns, e := resolve("net", f.Name()); e == nil {
				fmt.Println(ns.Id)
			} else {
				fmt.Printf("Can't read namespace from process %s due to %s\n", f.Name(), e)
			}
		}
	}
}

func listc() {
	fmt.Println("cgroups:")
}

func list() {
	if runtime.GOOS != "linux" {
		fmt.Println("Sorry, this is a Linux-specific tool.")
		os.Exit(1)
	}

	if lnamespaces {
		listn()
		return
	}
	if lcgroups {
		listc()
		return
	}
	listn()
	listc()
}

func main() {
	if lversion {
		about()
		os.Exit(0)
	}
	list()
}
