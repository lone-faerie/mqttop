package file

import (
	"os"

	"golang.org/x/sys/unix"
)

// Stat returns a FileInfo describing the named file.
func Stat(name string) (os.FileInfo, error) {
	name, err := abs(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(name)
}

// Exists reports whether the named file exists.
func Exists(name string) bool {
	_, err := Stat(name)
	return err == nil
}

// IsDir reports whether the named file is a directory.
func IsDir(name string) bool {
	info, err := Stat(name)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// AccessTime returns the last access time of the named file.
func AccessTime(name string) (sec, nsec int64, err error) {
	name, err = abs(name)
	if err != nil {
		return
	}
	var stat unix.Stat_t
	if err = unix.Stat(name, &stat); err != nil {
		return
	}
	sec, nsec = stat.Atim.Unix()
	return
}

// ModifyTime returns the last modify time of the named file.
func ModifyTime(name string) (sec, nsec int64, err error) {
	name, err = abs(name)
	if err != nil {
		return
	}
	var stat unix.Stat_t
	if err = unix.Stat(name, &stat); err != nil {
		return
	}
	sec, nsec = stat.Mtim.Unix()
	return
}

// ChangeTime returns the last change time of the named file.
func ChangeTime(name string) (sec, nsec int64, err error) {
	name, err = abs(name)
	if err != nil {
		return
	}
	var stat unix.Stat_t
	if err = unix.Stat(name, &stat); err != nil {
		return
	}
	sec, nsec = stat.Ctim.Unix()
	return
}

// Statfs returns information about the filesystem of the named file.
func Statfs(name string) (stat unix.Statfs_t, err error) {
	name, err = abs(name)
	if err != nil {
		return
	}
	err = unix.Statfs(name, &stat)
	return
}
