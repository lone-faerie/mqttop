package procfs

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/file"
)

const (
	MountPath = file.Separator + "proc"
)

const (
	cpuInfoPath    = MountPath + file.Separator + "cpuinfo"
	memInfoPath    = MountPath + file.Separator + "meminfo"
	fsPath         = MountPath + file.Separator + "filesystems"
	statPath       = MountPath + file.Separator + "stat"
	selfPath       = MountPath + file.Separator + "self"
	mountsPath     = MountPath + file.Separator + "1" + file.Separator + "mounts"
	selfMountsPath = selfPath + file.Separator + "mounts"
)

type (
	File = file.File
	Dir  = file.Dir
)

func Path(elem ...string) string {
	return MountPath + file.Separator + filepath.Join(elem...)
}

func CPUInfo() (*File, error) {
	return file.Open(cpuInfoPath)
}

func MemInfo() (*File, error) {
	return file.Open(memInfoPath)
}

func Stat() (*File, error) {
	return file.Open(statPath)
}

func Self() (*Dir, error) {
	return file.OpenDir(selfPath)
}

func SelfMounts() (*File, error) {
	return file.Open(selfMountsPath)
}

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

func Filesystems() (*File, error) {
	return file.Open(fsPath)
}
