//go:build !debug
package build

import "runtime/debug"

func load() {
	if pkg != "" && version != "" && buildTime != "" {
		version = semver(version)
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if pkg == "" {
		pkg = info.Main.Path
	}
	if version == "" {
		version = info.Main.Version
	}
	if buildTime == "" {
		for _, s := range info.Settings {
			if s.Key == "vcs.time" {
				buildTime = s.Value
				if buildTime[len(buildTime)-1] == 'Z' {
					buildTime = buildTime[:len(buildTime)-1] + "+00:00"
				}
				break
			}
		}
	}
}
