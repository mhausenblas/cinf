package main

import (
	"flag"
	"fmt"
	procfs "github.com/c9s/goprocinfo/linux"
	"log"
	"os"
	"runtime"
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

func listn() {
	fmt.Println("namespaces:")

	maxpid, err := procfs.ReadMaxPID("/proc/sys/kernel/pid_max")
	if err != nil {
		log.Fatal("Retrieving max process ID failed.")
	}

	pidlist, err := procfs.ListPID("/proc", maxpid)
	if err != nil {
		log.Fatal("Retrieving list of process IDs failed.")
	}

	for _, pid := range pidlist {
		proc, err := procfs.ReadProcess(pid, "/proc/")
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%v", proc)
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
