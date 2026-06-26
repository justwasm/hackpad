package main

import (
	"fmt"
	"os"
	"runtime"
)

func main() {
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("GOOS: %s\n", runtime.GOOS)
	fmt.Printf("GOARCH: %s\n", runtime.GOARCH)
	fmt.Printf("Compiler: %s\n", runtime.Compiler)
	fmt.Printf("NumCPU: %d\n", runtime.NumCPU())
	fmt.Printf("GOROOT: %s\n", runtime.GOROOT())
	fmt.Printf("Hostname: %s\n", hostnameOr("unknown"))
	fmt.Printf("Args: %v\n", os.Args)
	fmt.Printf("PATH: %s\n", os.Getenv("PATH"))
	fmt.Printf("HOME: %s\n", os.Getenv("HOME"))
	fmt.Printf("USER: %s\n", os.Getenv("USER"))
	wd, _ := os.Getwd()
	fmt.Printf("CWD: %s\n", wd)
}

func hostnameOr(fallback string) string {
	h, err := os.Hostname()
	if err != nil {
		return fallback
	}
	return h
}
