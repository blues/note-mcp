package main

import "runtime/debug"

// version is injected at build time via -ldflags "-X main.version=<semver>".
// It is empty for local builds (e.g. `go run`), in which case serverVersion()
// falls back to the VCS revision recorded in the build info.
var version string

// serverVersion returns the semantic version injected at build time, falling
// back to the git commit revision when no version was injected.
func serverVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return "unknown"
}
