package procfs

import (
	"bytes"
	"io"
	"sync"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
	"github.com/lone-faerie/mqttop/log"
)

// Mount describes a mount according to /proc/1/mounts (or /proc/self/mounts)
type Mount struct {
	Dev    string
	Mnt    string
	FSType string
}

var (
	nodev    = []byte("nodev")
	squashfs = []byte("squashfs")
	nullfs   = []byte("nullfs")
)

func validFSTypes() (map[string]bool, error) {
	f, err := Filesystems()
	if err != nil {
		return nil, err
	}

	defer f.Close()

	fstypes := map[string]bool{
		"zfs":   true,
		"wslfs": true,
		"drvfs": true,
	}

	for {
		line, err := f.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		if bytes.HasPrefix(line, nodev) {
			continue
		}

		line = bytes.TrimSpace(line)
		if byteutil.Equal(line, squashfs) || byteutil.Equal(line, nullfs) {
			continue
		}

		fstypes[string(line)] = true
	}

	return fstypes, nil
}

const fstabPath = file.Separator + "etc" + file.Separator + "fstab"

var (
	fstab     map[string]bool
	fstabTime struct {
		Sec  int64
		Nsec int64
	}
	fstabMu sync.Mutex
)

var (
	noneMnt = []byte("none")
	swapMnt = []byte("swap")
)

func fstabDisks() error {
	fstabMu.Lock()
	defer fstabMu.Unlock()

	sec, nsec, err := file.ModifyTime(fstabPath)
	if err != nil {
		return err
	}

	if fstabTime.Sec == sec && fstabTime.Nsec == nsec {
		return nil
	}

	fstabTime = struct {
		Sec  int64
		Nsec int64
	}{sec, nsec}

	if fstab == nil {
		fstab = make(map[string]bool)
	} else {
		clear(fstab)
	}

	f, err := file.Open(fstabPath)
	if err != nil {
		return err
	}

	defer f.Close()

	for {
		line, err := f.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if len(line) == 0 || line[0] == '#' {
			continue
		}

		_, line = byteutil.Column(line)
		mnt, _ := byteutil.Column(line)

		if len(mnt) == 0 || byteutil.Equal(mnt, noneMnt) || byteutil.Equal(mnt, swapMnt) {
			continue
		}

		fstab[string(mnt)] = true
	}

	log.Debug("procfs.MountInfo", "fstab", fstab)

	return nil
}

func findMounts(search map[string]*Mount, valid map[string]bool, useFSTab bool) error {
	if useFSTab {
		fstabMu.Lock()
		defer fstabMu.Unlock()
	}

	f, err := Mounts()
	if err != nil {
		return err
	}

	defer f.Close()

	var (
		cols             int
		dev, mnt, fstype []byte
	)

	for {
		line, err := f.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Debug("findMounts", err)
			return err
		}

		cols, _ = byteutil.Columns(line, &dev, &mnt, &fstype)
		if cols < 3 {
			continue
		}

		info := &Mount{
			Dev:    string(dev),
			Mnt:    string(mnt),
			FSType: string(fstype),
		}

		log.Debug("findMounts", "mnt", info.Mnt, "matchFSTab", useFSTab && fstab[info.Mnt], "matchValid", !useFSTab && valid[info.FSType])

		if (useFSTab && fstab[info.Mnt]) || (!useFSTab && valid[info.FSType]) {
			log.Debug("Found disk", "mnt", info.Mnt)
			search[info.Mnt] = info
		}
	}

	return nil
}

// MountInfo returns the disks mounted on the system, mapped by their mounting point.
// If useFSTab is true, the disk must be in /etc/fstab to be included.
func MountInfo(useFSTab bool) (map[string]*Mount, error) {
	valid, err := validFSTypes()
	if err != nil {
		return nil, err
	}

	log.Debug("procfs.MountInfo", "validFSTypes", valid)

	if useFSTab {
		if err = fstabDisks(); err != nil {
			return nil, err
		}
	}

	search := make(map[string]*Mount)

	if err = findMounts(search, valid, useFSTab); err != nil {
		return nil, err
	}

	return search, nil
}
