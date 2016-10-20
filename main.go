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
	VERSION = "0.2.0"
)

var (
	DEBUG    bool
	lversion bool
	targetns string
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
}

func main() {
	namespaces.DEBUG = DEBUG
	debug("=== SHOWING DEBUG MESSAGES ===")
	if lversion {
		about()
		os.Exit(0)
	}
	namespaces.Gather()
	if targetns != "" { // target a specific namespace
		namespaces.Show(targetns)
	} else {
		namespaces.Showall()
	}
}
