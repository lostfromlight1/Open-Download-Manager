// Package version holds build identity constants shared by the CLI and apps.
//
// Small packages like this avoid duplicating "what product/version is this?"
// across cmd/odm, the desktop shell, and future services.
package version

// Version is the application version for CLI and packaging metadata.
//
// In Go, a name that starts with an Uppercase letter is exported: other
// packages can read version.Version. A lowercase name would be package-private.
const Version = "0.5.0"

// ProductName is the user-facing product name.
const ProductName = "ODM — Open Download Manager"

// String returns a short identity line for the CLI.
//
// Methods and functions with Uppercase names are also exported.
func String() string {
	// + concatenates strings. For many pieces, prefer fmt.Sprintf.
	return ProductName + " " + Version
}
