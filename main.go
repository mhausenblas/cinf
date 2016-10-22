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
	VERSION = "0.3.0"
)

var (
	DEBUG     bool
	version   bool
	targetns  string
	targetpid string
	targetcg  string
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
	flag.StringVar(&targetcg, "cgroup", "", "List details of a cgroup a process belongs to. Format is CGROUP_HIERARCHY:PID, for example 2:1000.")

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
	case targetns != "": // we have a -namespace flag
		namespaces.LookupNS(targetns)
	case targetpid != "": // we have a -pid flag
		namespaces.LookupPID(targetpid)
	case targetcg != "": // we have a -cgroup flag
		namespaces.LookupCG(targetcg)
	default: // list all active namespaces
		namespaces.Showall()
	}
}
