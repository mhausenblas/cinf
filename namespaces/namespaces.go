package namespaces

import (
	"errors"
	"fmt"
	tw "github.com/olekukonko/tablewriter"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
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
	Uids    string
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
	processes       map[string][]Namespace
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
	// for all default operations and lookups:
	namespaces = make(map[Namespace][]Process)
	// for lookups only (PID -> namespaces):
	processes = make(map[string][]Namespace)
	MAX_COMMAND_LEN = 20
}

func contains(s int, slist []int) bool {
	for _, b := range slist {
		if b == s {
			return true
		}
	}
	return false
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
				k := strings.Split(l, ":")[0]
				v := strings.TrimSpace(strings.Split(l, ":")[1])
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
				case "Uid":
					// Uid:	1000	1000	1000	1000
					p.Uids = v
				}
			}
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

func usage(cg string, pid string) (map[string]string, error) {
	base := "/sys/fs/cgroup/"
	for _, ns := range processes[pid] { // looking up namespaces of process
		debug("checking namespace " + ns.Id)
		for _, p := range namespaces[ns] { // checking to find process details
			debug("checking process " + p.Pid)
			if pid == p.Pid {
				cgroups := p.Cgroups
				lines := strings.Split(cgroups, "\n")
				for _, l := range lines {
					debug("line " + l)
					if l != "" {
						chierarchy := strings.Split(l, ":")[0]
						cname := strings.Split(l, ":")[1]
						cpath := strings.Split(l, ":")[2]
						if cg == chierarchy { // matches targeted cgroup
							cdir := filepath.Join(base, cname, cpath)
							cfiles, _ := ioutil.ReadDir(cdir)
							cmap := make(map[string]string)
							for _, f := range cfiles { // read out values per control file
								cfname := filepath.Join(cdir, f.Name())
								// note that in the following we're ignoring write-only files:
								if cvalue, err := ioutil.ReadFile(cfname); err == nil {
									cmap[f.Name()] = string(cvalue)
								}
							}
							return cmap, nil
						}
					}
				}
			}
		}
	}
	return nil, errors.New(fmt.Sprintf("No control files found for cgroup %s of process %s", cg, pid))
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
	for _, f := range fn { // each file representing a process
		_, pid := filepath.Split(f)
		debug("looking at process: " + pid)
		// establishing mappings:
		for _, tns := range NS {
			debug("for namespace: " + string(tns))
			if ns, e := resolve(tns, pid); e == nil {
				p, _ := status(pid)
				// (namespace -> list of processes) mapping:
				namespaces[*ns] = append(namespaces[*ns], *p)
				// (process -> list of namespaces) mapping:
				processes[pid] = append(processes[pid], *ns)
			} else {
				debug(fmt.Sprintf("%s of process %s", e, pid))
			}
		}
	}
}

// LookupCG displays details about a cgroup a process belongs to.
// Note that cgofp is expected to be in the format CGROUP_HIERARCHY:PID
// with allowed values for CGROUP_HIERARCHY being the cgroups v1 hierarchy
// values as found in /proc/groups - for more infos see also
// http://man7.org/linux/man-pages/man7/cgroups.7.html
// For example:
//  namespaces.LookupCG("2:1000")
func LookupCG(cgofp string) {
	rp := regexp.MustCompile("([0-9])+:([0-9])+")
	if rp.MatchString(cgofp) { // the provided argument matches the expected format
		cg := strings.Split(cgofp, ":")[0]
		pid := strings.Split(cgofp, ":")[1]
		debug(fmt.Sprintf("Looking up cgroup %s of process %s", cg, pid))
		if cm, err := usage(cg, pid); err == nil {
			ptable := tw.NewWriter(os.Stdout)
			ptable.SetHeader([]string{"CONTROLFILE", "VALUE"})
			ptable.SetCenterSeparator("")
			ptable.SetColumnSeparator("")
			ptable.SetRowSeparator("")
			ptable.SetAlignment(tw.ALIGN_LEFT)
			ptable.SetHeaderAlignment(tw.ALIGN_LEFT)
			debug("\n\n=== SUMMARY")
			debug(fmt.Sprintf("control files: %v", cm))
			for cf, v := range cm {
				row := []string{cf, v}
				ptable.Append(row)
			}
			ptable.Render()
		} else {
			fmt.Println(err)
		}
	} else {
		fmt.Println("Provided argument is not in expected format. It should be CGROUP_HIERARCHY:PID.")
		fmt.Println("For example: 2:1000 to list details of cgroup with hierarchy ID 2 the process with PID 1000 belongs to.")
	}
}

// LookupPID displays the namespaces a process is in.
// For example:
//  namespaces.LookupPID("1000")
func LookupPID(pid string) {
	ptable := tw.NewWriter(os.Stdout)
	ptable.SetHeader([]string{"NAMESPACE", "TYPE"})
	ptable.SetCenterSeparator("")
	ptable.SetColumnSeparator("")
	ptable.SetRowSeparator("")
	ptable.SetAlignment(tw.ALIGN_LEFT)
	ptable.SetHeaderAlignment(tw.ALIGN_LEFT)
	debug("\n\n=== SUMMARY")

	for _, ns := range processes[pid] {
		debug("for namespace " + ns.Id)
		row := []string{ns.Id, string(ns.Type)}
		ptable.Append(row)
	}
	ptable.Render()

}

// LookupNS displays details about a specific namespace.
// For example:
//  namespaces.LookupNS("4026532198")
func LookupNS(targetns string) {
	ptable := tw.NewWriter(os.Stdout)
	ptable.SetHeader([]string{"PID", "PPID", "NAME", "CMD", "NTHREADS", "CGROUPS", "STATE"})
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
			// rendering process command line:
			cmd := p.Command
			if len(cmd) > MAX_COMMAND_LEN {
				cmd = cmd[:MAX_COMMAND_LEN]
			}
			row := []string{string(p.Pid), string(p.PPid), p.Name, cmd, string(p.Threads), p.Cgroups, p.State}
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
	ntable.SetHeader([]string{"NAMESPACE", "TYPE", "NPROCS", "USER", "CMD"})
	ntable.SetCenterSeparator("")
	ntable.SetColumnSeparator("")
	ntable.SetRowSeparator("")
	ntable.SetAlignment(tw.ALIGN_LEFT)
	ntable.SetHeaderAlignment(tw.ALIGN_LEFT)
	debug("\n\n=== SUMMARY")
	for n, pl := range namespaces {
		debug(fmt.Sprintf("namespace %s: %v\n", n.Id, pl))
		u := ""
		suids := make([]int, 0)
		for _, p := range pl {
			// using the effective UID here (which is the 2nd in the list):
			uid, _ := strconv.Atoi(strings.Fields(p.Uids)[1])
			if !contains(uid, suids) {
				suids = append(suids, int(uid))
			}
		}
		sort.Ints(suids)
		for _, uid := range suids {
			u += fmt.Sprintf("%d,", uid)
		}
		if strings.HasSuffix(u, ",") {
			u = u[0 : len(u)-1]
		}
		// rendering process command line:
		cmd := pl[0].Command
		if len(cmd) > MAX_COMMAND_LEN {
			cmd = cmd[:MAX_COMMAND_LEN]
		}
		// assembling one row (one namespace rendering)
		row := []string{string(n.Id), string(n.Type), strconv.Itoa(len(pl)), u, cmd}
		ntable.Append(row)
	}
	ntable.Render()
}
