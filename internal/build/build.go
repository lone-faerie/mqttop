// Package build provides varaibles that are set at build-time
// with the -X ldflag. If the values are not given at build-time,
// they will be determined from [debug.BuildInfo].
package build

import (
	"log"
	"regexp"
	"runtime/debug"
	"sync"
)

var (
	pkg       string
	version   string
	buildTime string
)

var once sync.Once

func semver(v string) string {
	loc := regexp.MustCompile(`v?\d+(\.\d+){0,2}`).FindStringIndex(v)
	if loc == nil {
		return v
	}
	return v[loc[0]:loc[1]]
}

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
		log.Println("Using debug.ReadBuildInfo() for pkg")
		pkg = info.Main.Path
	}
	if version == "" {
		log.Println("Using debug.ReadBuildInfo() for version")
		version = info.Main.Version
	}
	if buildTime == "" {
		log.Println("Using debug.ReadBuildInfo for build time")
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

func Package() string {
	once.Do(load)
	return pkg
}

func Version() string {
	once.Do(load)
	return version
}

func BuildTime() string {
	once.Do(load)
	return buildTime
}
