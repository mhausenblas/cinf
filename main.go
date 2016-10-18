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

type Process struct {
	Pid     uint64
	PPid    uint64
	Name    string
	State   string
	Threads int
}

var (
	lversion   bool
	namespaces map[Namespace]Process
)

func debug(m string) {
	if DEBUG {
		fmt.Printf("DEBUG: %s\n", m)
	}
}

func about() {
	fmt.Printf(BANNER)
	fmt.Printf("\nThis is cinf in version %s\n", VERSION)
	fmt.Print("See also https://github.com/mhausenblas/cinf\n\n")
}

func init() {
	flag.BoolVar(&lversion, "version", false, "List info about cinf, including its version")

	flag.Usage = func() {
		fmt.Printf("Usage: %s [args]\n\n", os.Args[0])
		fmt.Println("Arguments:")
		flag.PrintDefaults()
	}
	flag.Parse()

	namespaces = make(map[Namespace]Process)
}

func (nstype NSTYPE) String() string {
	switch nstype {
	case NS_MOUNT:
		return "mnt"
	case NS_UTS:
		return "uts"
	case NS_IPC:
		return "ipc"
	case NS_PID:
		return "pid"
	case NS_NET:
		return "net"
	case NS_USER:
		return "user"
	case NS_CGROUP:
		return "cgroup"
	}
	return ""
}

func resolve(nstype NSTYPE, pid uint64) (*Namespace, error) {
	nsfile := filepath.Join("/proc", string(pid), "ns", string(nstype))
	debug(nsfile)
	if content, err := os.Readlink(nsfile); err == nil {
		debug(content)
		// turn something like user:[4026531837] into 4026531837
		nsnum := strings.Split(content, ":")[1]
		nsnum = nsnum[1 : len(nsnum)-1]
		ns := Namespace{}
		ns.Type = nstype
		ns.Id = string(nsnum)
		return &ns, nil
	} else {
		return nil, err
	}
}

func status(pid uint64) (*Process, error) {
	sfile := filepath.Join("/proc", string(pid), "status")
	if s, err := ioutil.ReadFile(sfile); err == nil {
		p := Process{}
		lines := strings.Split(string(s), "\n")
		for _, l := range lines {
			k, v := strings.Split(l, ":")[0], strings.Trim(strings.Split(l, ":")[1], " ")
			switch k {
			case "Pid":
				ipid, _ := strconv.Atoi(v)
				p.Pid = uint64(ipid)
			case "PPid":
				ippid, _ := strconv.Atoi(v)
				p.PPid = uint64(ippid)
			case "Name":
				p.Name = v
			case "State":
				p.State = v
			case "Threads":
				it, _ := strconv.Atoi(v)
				p.Threads = it
			}
		}
		return &p, nil
	} else {
		return nil, err
	}
}

func list() {
	if runtime.GOOS != "linux" {
		fmt.Println("Sorry, this is a Linux-specific tool.")
		os.Exit(1)
	}
	files, _ := ioutil.ReadDir("/proc")
	for _, f := range files {
		if pid, err := strconv.Atoi(f.Name()); err == nil {
			if ns, e := resolve(NS_NET, uint64(pid)); e == nil {
				fmt.Println(ns.Id)
				p, _ := status(uint64(pid))
				namespaces[*ns] = *p
			} else {
				fmt.Printf("Can't read namespace from process %s due to %s\n", f.Name(), e)
			}
		}
	}
}

func main() {
	if lversion {
		about()
		os.Exit(0)
	}
	list()
}
