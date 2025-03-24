package file

import (
	"os"

	"golang.org/x/sys/unix"
)

func Stat(name string) (os.FileInfo, error) {
	name, err := abs(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(name)
}

func Exists(name string) bool {
	_, err := Stat(name)
	return err == nil
}

func IsDir(name string) bool {
	info, err := Stat(name)
	if err != nil {
		return false
	}
	return info.IsDir()
}

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

func Statfs(name string) (stat unix.Statfs_t, err error) {
	name, err = abs(name)
	if err != nil {
		return
	}
	err = unix.Statfs(name, &stat)
	return
}
