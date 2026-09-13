// Package version exposes the version of the tool that imports it.
//
// Every katbyte tool reports a version the same way: stamped by the linker at
// release time, falling back to the module version Go records when the tool
// was installed with `go install ...@version`, and "dev" for a plain local
// build. Keeping that logic here means each tool's goreleaser and makefile
// stamp the same two variables:
//
//	-X github.com/katbyte/go-kt/version.Version={{ .Tag }}
//	-X github.com/katbyte/go-kt/version.GitCommit={{ .ShortCommit }}
//
// The variables live in this module, but ReadBuildInfo reports the *main*
// module's version, so a tool built from go-kt still reports its own tag.
package version

import "runtime/debug"

// Version is the release version. It is set at build time with -X, otherwise
// taken from the main module's build info, otherwise "dev". It is never blank:
// an -X flag given an empty value has happened before and must not produce a
// tool that reports no version at all.
var Version = "dev"

// GitCommit is the short commit hash the binary was built from, set at build
// time with -X. It is empty for builds that do not stamp it, which is fine:
// it only ever decorates the version string.
var GitCommit string

// init runs the fallback once, at process start, so callers can read Version
// as a plain variable without a resolve step.
func init() {
	// ReadBuildInfo returns nil info when the binary was built without module
	// support, which resolve treats the same as an unknown version
	info, _ := debug.ReadBuildInfo()
	Version = resolve(Version, info)
}

// resolve implements the precedence: a stamped version wins, then the main
// module's version from build info (nil when there is none), then "dev". It
// is a pure function so the rules can be tested without a real build.
func resolve(stamped string, info *debug.BuildInfo) string {
	// an -X flag given an empty value leaves this empty rather than "dev", so
	// treat both as unset: a build should never report a blank version
	if stamped != "" && stamped != "dev" {
		return stamped
	}

	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}
