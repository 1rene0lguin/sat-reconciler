package main

import (
	"fmt"
	"runtime"
)

// Variables inyectadas en compilación (ldflags)
var (
	Version = "dev"
	Commit  = "none"
)

func main() {
	fmt.Println("🐺 Laura Inc. - SAT Reconciler Engine")
	fmt.Printf("Version: %s | Commit: %s\n", Version, Commit)
	fmt.Printf("Running on: %s/%s\n", runtime.GOOS, runtime.ARCH)
	
	fmt.Println("System ready. Waiting for commands...")
}
