// Package procfs provides access to various files in /proc
package procfs

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/file"
)

const (
	MountPath = file.Separator + "proc" // /proc
)

const (
	cpuInfoPath    = MountPath + file.Separator + "cpuinfo"                       // /proc/cpuinfo
	memInfoPath    = MountPath + file.Separator + "meminfo"                       // /proc/meminfo
	fsPath         = MountPath + file.Separator + "filesystems"                   // /proc/filesystems
	statPath       = MountPath + file.Separator + "stat"                          // /proc/stat
	selfPath       = MountPath + file.Separator + "self"                          // /proc/self
	mountsPath     = MountPath + file.Separator + "1" + file.Separator + "mounts" // /proc/1/mounts
	selfMountsPath = selfPath + file.Separator + "mounts"                         // /proc/self/mounts
)

type (
	File = file.File
	Dir  = file.Dir
)

func Path(elem ...string) string {
	return MountPath + file.Separator + filepath.Join(elem...)
}

// CPUInfo returns the file /proc/cpuinfo
func CPUInfo() (*File, error) {
	return file.Open(cpuInfoPath)
}

// MemInfo returns the file /proc/meminfo
func MemInfo() (*File, error) {
	return file.Open(memInfoPath)
}

// Stat returns the file /proc/stat
func Stat() (*File, error) {
	return file.Open(statPath)
}

// Self returns the directory /proc/self
func Self() (*Dir, error) {
	return file.OpenDir(selfPath)
}

// SelfMounts returns the file /proc/self/mounts
func SelfMounts() (*File, error) {
	return file.Open(selfMountsPath)
}

// Mounts returns the file /proc/1/mounts, or /proc/self/mounts if
// /proc/1/mounts cannot be opened
func Mounts() (*File, error) {
	f, err := file.Open(mountsPath)
	if err == nil {
		return f, err
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, fs.ErrPermission) {
		f, err = file.Open(selfMountsPath)
	}
	return f, err
}

// Filesystems returns the file /proc/filesystems
func Filesystems() (*File, error) {
	return file.Open(fsPath)
}
