//go:build darwin
// +build darwin

package main

import "fmt"

// macOS stub - libfido2 CGO linking issues in CI
func main() {
	fmt.Println("fit: Hardware CLI has libfido2 linking issues on macOS in CI")
	fmt.Println("For local development, install libfido2 and build with 'go build -tags linux'")
}
