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
