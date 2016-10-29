package main

import (
	"flag"
	"fmt"
	"github.com/mhausenblas/cinf/namespaces"
	"os"
	"strconv"
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
	VERSION = "0.5.0"
)

var (
	DEBUG     bool
	version   bool
	targetns  string
	targetpid string
	cgspec    string
	monspec   string
	logspec   string
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
	flag.BoolVar(&version, "version", false, "List info about cinf, including its version.")
	flag.StringVar(&targetns, "namespace", "", "List details about namespace with provided ID. You can get the namespace ID by running cinf without arguments.")
	flag.StringVar(&targetpid, "pid", "", "List namespaces the process with provided process ID is in.")
	flag.StringVar(&cgspec, "cgroup", "", "List details of a cgroup a process belongs to. Format is PID:CGROUP_HIERARCHY, for example 1000:2.")
	flag.StringVar(&monspec, "mon", "", "Monitor process with provided process ID/control file(s). Format is PID:CF1,CF2,… for example 1000:memory.usage_in_bytes")
	flag.StringVar(&logspec, "log", "", "Continuously output namespace and cgroups metrics. Format is OUTPUT_DEF:INTERVAL, for example JSON:0.2 or SYSLOG:5")

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
}

func main() {
	namespaces.DEBUG = DEBUG
	debug("=== SHOWING DEBUG MESSAGES ===")
	if version {
		about()
		os.Exit(0)
	}
	namespaces.Gather()

	switch {
	case targetns != "": // we have a --namespace flag
		namespaces.LookupNS(targetns)
	case targetpid != "": // we have a --pid flag
		namespaces.LookupPID(targetpid)
	case cgspec != "": // we have a --cgroup flag
		namespaces.LookupCG(cgspec)
	case monspec != "": // we have a --mon flag
		namespaces.MonitorPID(monspec)
	case logspec != "": // we have a --log flag
		namespaces.DoMetrics(logspec)
	default: // list all active namespaces
		namespaces.Showall()
	}
}
