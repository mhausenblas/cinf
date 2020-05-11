package namespaces

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	tm "github.com/buger/goterm"
	tw "github.com/olekukonko/tablewriter"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"syscall"
	"unsafe"
)

type NSTYPE string

type Namespace struct {
	Type NSTYPE
	Id   string
}

type Process struct {
	Pid     string `json:"pid"`
	PPid    string `json:"ppid"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Threads string `json:"nthreads"`
	Cgroups string `json:"cgroups"`
	Uids    string `json:"uids"`
	Command string `json:"cmd"`
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
	availablecgs    map[string]string
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
	// maps cgroup names to hierarchy IDs:
	availablecgs = make(map[string]string)
	MAX_COMMAND_LEN = int(getWidth()) -70
}

// Get terminal width, thanks to:
// https://stackoverflow.com/a/16576712/9979477
func getWidth() uint {
        type winsize struct {
                Row    uint16
                Col    uint16
                Xpixel uint16
                Ypixel uint16
        }

        ws := &winsize{}
        retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
        uintptr(syscall.Stdin),
        uintptr(syscall.TIOCGWINSZ),
        uintptr(unsafe.Pointer(ws)))

    if int(retCode) == -1 {
        panic(errno)
    }
    return uint(ws.Col)
}

func contains(s int, slist []int) bool {
	for _, b := range slist {
		if b == s {
			return true
		}
	}
	return false
}

// initcgs reads out available cgroups to map
// from cgroup name to hierarchy ID
func initcgs() {
	acgs := "/proc/cgroups"
	availablecgs = map[string]string{}
	if c, err := ioutil.ReadFile(acgs); err == nil {
		lines := strings.Split(string(c), "\n")
		for _, l := range lines {
			if l != "" && !strings.Contains(l, "#") { // ignore header
				name := strings.Fields(l)[0]
				id := strings.Fields(l)[1]
				enabled := strings.Fields(l)[3]
				if enabled == "1" {
					availablecgs[name] = id
				}
			}
		}
	}
	debug(fmt.Sprintf("available cgroups: %v", availablecgs))
}

// Turn zero bytes in given slice into ASCII spaces
func zeros_to_spaces(arr []byte) []byte{
    for idx, value := range arr {
        if value == 0x00 {
                arr[idx] = 0x20
                }
        }
    return arr
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
					//      real, effective, saved set, filesystem
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
			p.Command = strings.TrimSpace(string(zeros_to_spaces(cmd)))
		}
		return &p, nil
	} else {
		return nil, err
	}
}

// lprocess returns the process with the pid provided.
// For example:
//  namespaces.lprocess("1234")
func lprocess(pid string) *Process {
	for _, ns := range processes[pid] { // looking up namespaces of process
		debug("checking namespace " + ns.Id)
		for _, p := range namespaces[ns] { // checking to find process details
			debug("checking process " + p.Pid)
			if pid == p.Pid {
				return &p
			}
		}
	}
	return nil
}

// usage reads out values of control files under /sys/fs/cgroup/
// given a process pid and a cgroup hierarchy ID cg.
// For example:
//  namespaces.usage("1234", "5")
func usage(pid string, cg string) (map[string]string, error) {
	base := "/sys/fs/cgroup/"
	p := lprocess(pid)

	if p == nil {
		fmt.Fprintln(os.Stderr, "PID not found.")
		os.Exit(1)
	}

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
	return nil, errors.New(fmt.Sprintf("No control files found for cgroup %s of process %s", cg, pid))
}

///////////////////////////////////////////////////////////////////////////////
// PUBLIC API
//

// Gather reads out process-related info from /proc and fills the global
// namespaces map with it. Note that only filenames that match the [0-9]* pattern
// are considered here since those are the ones representing processes, with
// the filename being the PID.
//
// Example:
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
	initcgs()
}

// LookupCG displays details about a cgroup a process belongs to.
// Note that cgspec is expected to be in the format PID:CGROUP_HIERARCHY
// with allowed values for CGROUP_HIERARCHY being the cgroups v1 hierarchy
// values as found in /proc/groups - see http://man7.org/linux/man-pages/man7/cgroups.7.html
// for more infos.
//
// Example:
//  namespaces.LookupCG("1000:2")
func LookupCG(cgspec string) {
	rp := regexp.MustCompile("([0-9])+:([0-9])+")
	if rp.MatchString(cgspec) { // the provided argument matches expected format
		pid := strings.Split(cgspec, ":")[0]
		cg := strings.Split(cgspec, ":")[1]
		debug(fmt.Sprintf("Looking up cgroup %s of process %s", cg, pid))
		if cm, err := usage(pid, cg); err == nil {
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
		fmt.Println("Provided argument is not in expected format. It should be PID:CGROUP_HIERARCHY")
		fmt.Println("For example: 1000:2 lists details of cgroup with hierarchy ID 2 the process with PID 1000 belongs to.")
	}
}

// LookupPID displays the namespaces a process is in.
//
// Example:
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
//
// Example:
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

// MonitorPID monitors control files values of cgroup a process belongs to.
// Note that monspec is expected to be in the format PID:CF1,CF2,…
// with allowed values for CFx being the cgroups control files as described
// in http://man7.org/linux/man-pages/man7/cgroups.7.html
//
// Example:
//  namespaces.MonitorPID("1000:memory.usage_in_bytes,memory.max_usage_in_bytes")
func MonitorPID(monspec string) {
	rp := regexp.MustCompile("([0-9])+:*")
	if rp.MatchString(monspec) { // the provided argument matches expected format
		pid := strings.Split(monspec, ":")[0]
		colspec := strings.Split(monspec, ":")[1]
		columns := strings.Split(colspec, ",")
		p, _ := status(pid)
		debug(fmt.Sprintf("Monitoring process %s with column spec %s", pid, colspec))
		tm.Clear()
		for {
			tm.MoveCursor(1, 1)
			nsl := processes[pid]
			tm.Printf("cinf ")
			tm.Printf(tm.Background(tm.Color(fmt.Sprintf("PID [%s] PPID [%s] CMD [%s]", p.Pid, p.PPid, p.Command), tm.BLACK), tm.WHITE))
			tm.MoveCursor(2, 1)
			tm.Printf("     UIDS [%s] STATE [%s]", p.Uids, p.State)
			tm.MoveCursor(3, 1)
			tm.Printf("     NAMESPACES [%v]", nsl)
			tm.MoveCursor(5, 1)
			// formatting see https://golang.org/pkg/text/tabwriter/#Writer.Init
			cftable := tm.NewTable(5, 10, 5, ' ', 0)
			fmt.Fprintf(cftable, "CONTROLFILE\tVALUE\n")
			for _, c := range columns { // each column (= CF) in the spec
				// map CF -> cgroup -> cgroup hierarchy ID:
				cgname := strings.Split(string(c), ".")[0]
				cgid := availablecgs[cgname]
				if cm, err := usage(pid, cgid); err == nil {
					for cf, v := range cm {
						if cf == c { // only show selected as per colspec
							fmt.Fprintf(cftable, "%s\t%s\n", cf, strings.Replace(v, "\n", " ", -1))
						}
					}
				}
			}
			tm.Println(cftable)
			tm.Flush()
			time.Sleep(time.Second)
		}
	} else {
		fmt.Println("Provided argument is not in expected format. It should be PID:CONTROLFILE1,CONTROLFIL2,….")
		fmt.Println("For example: 1000:memory.usage_in_bytes lists details of memory.usage_in_bytes control file the process with PID 1000 belongs to.")
	}
}

// DoMetrics continuously outputs namespace and cgroups metrics.
// Note that logspec is expected to be in the format OUTPUT_DEF:INTERVAL,
// with allowed values for OUTPUT_DEF being RAW or HTTP and INTERVAL
// specified in milliseconds.
//
// Example:
//  namespaces.DoMetrics("RAW:1000")
func DoMetrics(logspec string) {
	rp := regexp.MustCompile("[:ascii:]*:([0-9])+")
	if rp.MatchString(logspec) { // the provided argument matches expected format
		od := strings.Split(logspec, ":")[0]
		interval, _ := strconv.Atoi(strings.Split(logspec, ":")[1])
		var out bytes.Buffer
		for {
			fmt.Println("outputting to", od)
			for _, pl := range namespaces {
				for _, p := range pl {
					ep, _ := json.Marshal(p)
					json.Indent(&out, ep, "", "\t")
					out.WriteTo(os.Stdout)
				}
			}
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	} else {
		fmt.Println("Provided argument is not in expected format. It should be OUTPUT_DEF:INTERVAL")
		fmt.Println("For example: RAW:1000 will output all namespace and cgroups metrics to stdout, every second.")
	}
}

// Showall displays details about all active namespaces.
// For example:
//  namespaces.Showall()
func Showall() {
	ntable := tw.NewWriter(os.Stdout)
	ntable.SetHeader([]string{"NAMESPACE", "TYPE", "NPROCS", "USERS", "CMD"})
	ntable.SetCenterSeparator("")
	ntable.SetColumnSeparator("")
	ntable.SetRowSeparator("")
	ntable.SetAlignment(tw.ALIGN_LEFT)
	ntable.SetHeaderAlignment(tw.ALIGN_LEFT)
	ntable.SetAutoWrapText(false)
	debug("\n\n=== SUMMARY")

	keys := make([]Namespace, 0, len(namespaces))
		for k := range namespaces {
		keys = append(keys, k)
		}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Id < keys[j].Id
		})

	for _, n := range keys {
		pl := namespaces[n]

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
