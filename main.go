package main

import (
	"flag"
	"fmt"
	tw "github.com/olekukonko/tablewriter"
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
	Pid     string
	PPid    string
	Name    string
	State   string
	Threads string
	Cgroups string
	Uidmap  string
}

var (
	DEBUG      bool
	NS         []NSTYPE
	lversion   bool
	targetns   string
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

	if flag.NArg() == 1 { // we have a target namespace
		targetns = flag.Args()[0]
	}

	DEBUG = false
	if envd := os.Getenv("DEBUG"); envd != "" {
		if d, err := strconv.ParseBool(envd); err == nil {
			DEBUG = d
		}
	}
	// note: cgroups are not included in the following:
	NS = []NSTYPE{NS_MOUNT, NS_UTS, NS_IPC, NS_PID, NS_NET, NS_USER}
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
	// try to read out data about process status:
	if s, err := ioutil.ReadFile(sfile); err == nil {
		p := Process{}
		lines := strings.Split(string(s), "\n")
		for _, l := range lines {
			debug("status field " + l)
			if l != "" {
				k, v := strings.Split(l, ":")[0], strings.TrimSpace(strings.Split(l, ":")[1])
				switch k {
				case "Pid":
					p.Pid = v
				case "PPid":
					p.PPid = v
				case "Name":
					p.Name = v
				case "State":
					p.State = v
				case "Threads":
					p.Threads = v
				}
			}
		}
		// try to read out data about UIDs:
		uidmapfile := filepath.Join("/proc", pid, "uid_map")
		if uidmap, uerr := ioutil.ReadFile(uidmapfile); uerr == nil {
			p.Uidmap = strings.TrimSpace(string(uidmap))
		}
		// now try to read out data about cgroups:
		cfile := filepath.Join("/proc", pid, "cgroup")
		if cg, cerr := ioutil.ReadFile(cfile); cerr == nil {
			p.Cgroups = string(cg)
		}
		return &p, nil
	} else {
		return nil, err
	}
}

func gatherns() {
	if runtime.GOOS != "linux" {
		fmt.Println("Sorry, this is a Linux-specific tool.")
		os.Exit(1)
	}
	fn, _ := filepath.Glob("/proc/[0-9]*")
	for _, f := range fn {
		_, pid := filepath.Split(f)
		debug("looking at process: " + pid)
		for _, tns := range NS {
			debug("for namespace: " + string(tns))
			if ns, e := resolve(tns, pid); e == nil {
				p, _ := status(pid)
				namespaces[*ns] = append(namespaces[*ns], *p)
			} else {
				debug(fmt.Sprintf("%s of process %s", e, pid))
			}
		}
	}
}

func showns(target string) {
	ptable := tw.NewWriter(os.Stdout)
	ptable.SetHeader([]string{"PID", "PPID", "NAME", "STATE", "THREADS", "CGROUPS"})
	ptable.SetCenterSeparator("")
	ptable.SetColumnSeparator("")
	ptable.SetRowSeparator("")
	ptable.SetAlignment(tw.ALIGN_LEFT)
	ptable.SetHeaderAlignment(tw.ALIGN_LEFT)
	debug("\n\n=== SUMMARY")

	for _, tns := range NS {
		debug("for namespace " + string(tns))
		ns := Namespace{}
		ns.Type = tns
		ns.Id = target
		pl := namespaces[ns]
		for _, p := range pl {
			debug(fmt.Sprintf("looking in namespace %s at process %d\n", tns, p.Pid))
			row := []string{}
			row = []string{string(p.Pid), string(p.PPid), p.Name, p.State, string(p.Threads), p.Cgroups}
			ptable.Append(row)
		}
	}
	ptable.Render()
}

func showallns() {
	ntable := tw.NewWriter(os.Stdout)
	ntable.SetHeader([]string{"NAMESPACE", "TYPE", "NPROCS", "USER", "OUSER"})
	ntable.SetCenterSeparator("")
	ntable.SetColumnSeparator("")
	ntable.SetRowSeparator("")
	ntable.SetAlignment(tw.ALIGN_LEFT)
	ntable.SetHeaderAlignment(tw.ALIGN_LEFT)
	debug("\n\n=== SUMMARY")
	for n, pl := range namespaces {
		debug(fmt.Sprintf("namespace %s: %v\n", n.Id, pl))
		row := []string{}
		// note that the user listing here is really a short-cut, needs improvement:
		user := strings.Fields(pl[0].Uidmap)[0]
		ouser := strings.Fields(pl[0].Uidmap)[1]
		row = []string{string(n.Id), string(n.Type), strconv.Itoa(len(pl)), user, ouser}
		ntable.Append(row)
	}
	ntable.Render()
}

func main() {
	debug("=== SHOWING DEBUG MESSAGES ===")
	if lversion {
		about()
		os.Exit(0)
	}
	gatherns()
	if targetns != "" { // target a specific namespace
		showns(targetns)
	} else {
		showallns()
	}
}
