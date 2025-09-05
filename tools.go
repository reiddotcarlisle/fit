//go:build tools
// +build tools

// This file tracks tool dependencies (e.g., staticcheck) so that `go mod tidy`
// retains them in go.mod. These are not built into the final binaries.
package main

import _ "honnef.co/go/tools/cmd/staticcheck"
