package version

// Version is the canonical ndrop release version.
//
// Plain `go build` uses this value directly. The Makefile reads this same
// value for release artifact names and injects it into binaries with ldflags.
var Version = "1.0.0"
