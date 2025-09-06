//go:build windows
// +build windows

package main

import "fmt"

// Windows stub - libfido2 C headers not available in CI
func main() {
	fmt.Println("fit: Hardware CLI requires libfido2 (not available on Windows in CI)")
	fmt.Println("Use fit-hello for Windows Hello functionality")
}
