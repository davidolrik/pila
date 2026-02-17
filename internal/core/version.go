package core

import "runtime/debug"

// version is set via ldflags at build time:
//
// -ldflags "-X pila.olrik.dev/internal/core.version=1.2.3"
var version string

// Version is the resolved application version.
var Version = resolveVersion()

func resolveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "(devel)"
}
