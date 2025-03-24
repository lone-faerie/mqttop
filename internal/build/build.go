package build

import (
	"runtime/debug"
	"sync"
)

var (
	version string
	pkg     string
)

var once sync.Once

func load() {
	if version != "" && pkg != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "" {
		version = info.Main.Version
	}
	if pkg == "" {
		pkg = info.Main.Path
	}
}

func Version() string {
	once.Do(load)
	return version
}

func Package() string {
	once.Do(load)
	return pkg
}
