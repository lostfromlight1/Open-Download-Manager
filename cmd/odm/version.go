package main

import (
	"fmt"

	// This import path is our module path from go.mod plus the package folder.
	// internal/ packages can only be imported by code inside the same module,
	// which keeps helpers private to this repository.
	"github.com/lostfromlight1/Open-Download-Manager/internal/version"
)

// printVersion prints the product name and version string.
func printVersion() {
	// version.String() is a function in another package.
	// Exported names in Go start with an Uppercase letter (String, not string).
	fmt.Println(version.String())
}
