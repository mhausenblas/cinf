package namespaces

import (
	"fmt"
	tw "github.com/olekukonko/tablewriter"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type NSTYPE string

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
	Command string
}

// The supported namespaces
const (
	NS_MOUNT  NSTYPE = "mnt"    // CLONE_NEWNS, filesystem mount points
	NS_UTS    NSTYPE = "uts"    // CLONE_NEWUTS, nodename and NIS domain name
	NS_IPC    NSTYPE = "ipc"    // CLONE_NEWIPC, interprocess communication
	NS_PID    NSTYPE = "pid"    // CLONE_NEWPID, process ID number space isolation
	NS_NET    NSTYPE = "net"    // CLONE_NEWNET, network system resources
	NS_USER   NSTYPE = "user"   // CLONE_NEWUSER, user and group ID number space isolation
	NS_CGROUP NSTYPE = "cgroup" // CLONE_NEWCGROUP, cgroup root directory
)

var (
	DEBUG           bool
	NS              []NSTYPE
	namespaces      map[Namespace][]Process
	MAX_COMMAND_LEN int
)

func debug(m string) {
	if DEBUG {
		fmt.Printf("DEBUG: %s\n", m)
	}
}

func init() {
	// note: cgroups are not included in the following:
	NS = []NSTYPE{NS_MOUNT, NS_UTS, NS_IPC, NS_PID, NS_NET, NS_USER}
	namespaces = make(map[Namespace][]Process)
	MAX_COMMAND_LEN = 20
}

// resolve populates the specified namespace of a process.
// For example:
//  namespaces.resolve(NS_USER, "1234")
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

// status reads out process information from /proc/$PID/status.
// For example:
//  namespaces.status("1234")
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
		// this should really be read from /proc/$PID/status
		// Uid:	1000	1000	1000	1000
		uidmapfile := filepath.Join("/proc", pid, "uid_map")
		if uidmap, uerr := ioutil.ReadFile(uidmapfile); uerr == nil {
			p.Uidmap = strings.TrimSpace(string(uidmap))
		}
		// now try to read out data about cgroups:
		cfile := filepath.Join("/proc", pid, "cgroup")
		if cg, cerr := ioutil.ReadFile(cfile); cerr == nil {
			p.Cgroups = string(cg)
		}
		// try to read out process' command:
		cmdfile := filepath.Join("/proc", pid, "cmdline")
		if cmd, cerr := ioutil.ReadFile(cmdfile); cerr == nil {
			p.Command = strings.TrimSpace(string(cmd))
		}
		return &p, nil
	} else {
		return nil, err
	}
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC API
//

// Gather reads out process-related info from /proc and fills the global
// namespaces map with it. Note that only filenames that match the [0-9]* pattern
// are considered here since those are the ones representing processes, with
// the filename being the PID.
// For example:
//  namespaces.Gather()
func Gather() {
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

// Show displays details about a specific namespace.
// For example:
//  namespaces.Show("4026532198")
func Show(targetns string) {
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
		ns.Id = targetns
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

// Showall displays details about all active namespaces.
// For example:
//  namespaces.Showall()
func Showall() {
	ntable := tw.NewWriter(os.Stdout)
	ntable.SetHeader([]string{"NAMESPACE", "TYPE", "NPROCS", "USER", "OUSER", "CMD"})
	ntable.SetCenterSeparator("")
	ntable.SetColumnSeparator("")
	ntable.SetRowSeparator("")
	ntable.SetAlignment(tw.ALIGN_LEFT)
	ntable.SetHeaderAlignment(tw.ALIGN_LEFT)
	debug("\n\n=== SUMMARY")
	for n, pl := range namespaces {
		debug(fmt.Sprintf("namespace %s: %v\n", n.Id, pl))
		row := []string{}
		// rendering user and outside user:
		// picks UID of first process and indicates
		// how many more there are, if any
		user := ""
		ouser := ""
		uids := make(map[string]int)
		ouids := make(map[string]int)
		for _, p := range pl {
			uids[string(p.Uidmap[0])]++
			ouids[string(p.Uidmap[2])]++
		}
		for uid, uidcnt := range uids {
			user += fmt.Sprintf("%s (%d)", uid, uidcnt)
		}
		for ouid, ouidcnt := range uids {
			ouser += fmt.Sprintf("%s (%d)", ouid, ouidcnt)
		}

		// rendering process command line:
		cmd := pl[0].Command
		if len(cmd) > MAX_COMMAND_LEN {
			cmd = cmd[:MAX_COMMAND_LEN]
		}
		// assembling one row (one namespace rendering)
		row = []string{string(n.Id), string(n.Type), strconv.Itoa(len(pl)), user, ouser, cmd}
		ntable.Append(row)
	}
	ntable.Render()
}
