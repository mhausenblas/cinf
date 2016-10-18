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
)

type NSTYPE string

const (
	NS_MOUNT  NSTYPE = "mnt"    // CLONE_NEWNS, filesystem mount points
	NS_UTS    NSTYPE = "uts"    // CLONE_NEWUTS, nodename and NIS domain name
	NS_IPC    NSTYPE = "ipc"    // CLONE_NEWIPC, interprocess communication
	NS_PID    NSTYPE = "pid"    // CLONE_NEWPID, process ID number space isolation
	NS_NET    NSTYPE = "net"    // CLONE_NEWNET, network system resources
	NS_USER   NSTYPE = "user"   // CLONE_NEWUSER, user and group ID number space isolation
	NS_CGROUP NSTYPE = "cgroup" // CLONE_NEWCGROUP, cgroup root directory
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
	DEBUG      bool
	lversion   bool
	namespaces map[Namespace][]Process
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

	DEBUG = false
	if envd := os.Getenv("DEBUG"); envd != "" {
		if d, err := strconv.ParseBool(envd); err == nil {
			DEBUG = d
		}
	}

	namespaces = make(map[Namespace][]Process)
}

func resolve(nstype NSTYPE, pid string) (*Namespace, error) {
	debug("namespace type: " + string(nstype))
	nsfile := filepath.Join("/proc", pid, "ns", string(nstype))
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

func status(pid string) (*Process, error) {
	sfile := filepath.Join("/proc", pid, "status")
	debug("reading " + sfile)
	if s, err := ioutil.ReadFile(sfile); err == nil {
		p := Process{}
		lines := strings.Split(string(s), "\n")
		for _, l := range lines {
			debug("status field " + l)
			if l != "" {
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
	fn, _ := filepath.Glob("/proc/[0-9]*")
	for _, f := range fn {
		_, pid := filepath.Split(f)
		debug(pid)
		if ns, e := resolve(NS_NET, pid); e == nil {
			p, _ := status(pid)
			// if _, ok := namespaces[*ns]; ok { // we already have a process entry
			namespaces[*ns] = append(namespaces[*ns], *p)
			// } else { // init the process list
			// namespaces[*ns]
			// }
		} else {
			fmt.Printf("Can't read namespace from process %s due to %s\n", pid, e)
		}
	}
	debug("\n\n=== SUMMARY")
	for n, pl := range namespaces {
		debug(fmt.Sprintf("namespace %s: %v\n", n.Id, pl))
		fmt.Printf("%s (%d)\n", n.Id, len(pl))
	}
}

func main() {
	debug("=== SHOWING DEBUG MESSAGES ===")
	if lversion {
		about()
		os.Exit(0)
	}
	list()
}
