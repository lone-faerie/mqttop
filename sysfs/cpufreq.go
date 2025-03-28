package sysfs

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/lone-faerie/mqttop/internal/file"
)

type CPUFreq struct {
	Base int64
	curr int64
	Min  int64
	Max  int64
	Path string
}

func coreFreqs(found []string) ([]string, error) {
	d, err := CPU()
	if err != nil {
		return found, err
	}
	defer d.Close()
	err = d.WalkNames(func(name string) error {
		suffix, ok := strings.CutPrefix(name, "cpu")
		if !ok {
			return nil
		}
		id, err := strconv.Atoi(suffix)
		if err != nil {
			return nil
		}
		path := filepath.Join(cpuDevicesPath, name, "cpufreq")
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if n := id + 1; n > cap(found) {
			found = slices.Grow(found, n-cap(found))[:n]
		} else if n > len(found) {
			found = found[:n]
		}
		found[id] = path
		return nil
	})
	return found, err
}

func policyFreqs(found []string) ([]string, error) {
	d, err := file.OpenDir(filepath.Join(cpuDevicesPath, "cpufreq"))
	if err != nil {
		return nil, err
	}
	defer d.Close()
	err = d.WalkNames(func(name string) error {
		suffix, ok := strings.CutPrefix(name, "policy")
		if !ok {
			return nil
		}
		id, err := strconv.Atoi(suffix)
		if err != nil {
			return nil
		}
		path := filepath.Join(cpuDevicesPath, name)
		if n := id + 1; n > cap(found) {
			found = slices.Grow(found, n-cap(found))[:n]
		} else if n > len(found) {
			found = found[:n]
		}
		found[id] = path
		return nil
	})
	return found, err
}

func CPUFreqs() ([]CPUFreq, error) {
	found, err := coreFreqs(nil)
	if err != nil {
		return nil, err
	}
	if len(found) == 0 {
		found, err = policyFreqs(found)
		if err != nil {
			return nil, err
		}
	}
	if len(found) == 0 {
		return nil, nil
	}
	freqs := make([]CPUFreq, len(found))
	for i, dir := range found {
		base, err := file.ReadInt(filepath.Join(dir, "base_frequency"))
		if err != nil {
			return freqs, err
		}
		max, err := file.ReadInt(filepath.Join(dir, "scaling_max_freq"))
		if err != nil {
			continue
		}
		min, err := file.ReadInt(filepath.Join(dir, "scaling_min_freq"))
		if err != nil {
			continue
		}
		freqs[i] = CPUFreq{base, 0, min, max, filepath.Join(dir, "scaling_cur_freq")}
	}
	return freqs, nil
}

func (f *CPUFreq) Read() (int64, error) {
	v, err := file.ReadInt(f.Path)
	if err == nil {
		f.curr = v
	}
	return f.curr, err
}

func (f CPUFreq) Curr() int64 {
	return f.curr
}
