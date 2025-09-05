//go:build !windows
// +build !windows

package main

import "fmt"

// This stub allows the module to vet and test on non-Windows platforms in CI without
// pulling in winhello dependencies that have Windows-only constraints.
func main() {
	fmt.Println("fit-hello: Windows-only binary (stub on non-Windows platforms)")
}
