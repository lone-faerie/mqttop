// Package build provides varaibles that are set at build-time
// with the -X ldflag. If the values are not given at build-time,
// they will be determined from [debug.BuildInfo].
package build

import (
	"regexp"
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

// Package returns the current package. This is set by including
// the following ldflag:
//
//	-X 'github.com/lone-faerie/mqttop/internal/pkg=<package>'
//
// If the flag is not included, the value is loaded from [debug.ReadBuildInfo]
func Package() string {
	once.Do(load)
	return pkg
}

// Version returns the current version. This is set by including
// the following ldflag:
//
//	-X 'github.com/lone-faerie/mqttop/internal/version=<version>'
//
// If the flag is not included, the value is loaded from [debug.ReadBuildInfo]
// If the binary was built with the 'debug' tag, the value " (dev)" is appended
// to the version
func Version() string {
	once.Do(load)
	return version
}

// BuildTime returns the time the binary was built. This is set by including
// the following ldflag:
//
//	-X 'github.com/lone-faerie/mqttop/internal/buildTime=<build time>'
//
// If the flag is not included, the value is loaded from [debug.ReadBuildInfo]
func BuildTime() string {
	once.Do(load)
	return buildTime
}
