package sysfs

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lone-faerie/mqttop/internal/file"
)

type Sensor struct {
	Name  string
	Label string
	Path  string
	Max   int64
	value int64
}

func (s *Sensor) Read() (int64, error) {
	v, err := file.ReadInt(s.Path)
	if err == nil {
		s.value = v
	}
	return s.value, err
}

func (s *Sensor) Value() int64 {
	return s.value
}

func hwmonSensors(search map[string]bool) (gotCoretemp bool, err error) {
	d, err := HWMon()
	if err != nil {
		return
	}
	defer d.Close()
	err = d.WalkSymlinks(func(path string) error {
		if search[path] {
			return nil
		}
		if !gotCoretemp && strings.Contains(path, "coretemp") {
			gotCoretemp = true
		}
		files, err := file.ReadDirNames(path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if strings.HasPrefix(file, "temp") && strings.HasSuffix(file, "_input") {
				search[path] = true
				break
			}
		}
		return nil
	})
	return
}

func coretempSensors(search map[string]bool) (gotCoretemp bool, err error) {
	d, err := Coretemp()
	if err != nil {
		return
	}
	defer d.Close()
	err = d.WalkSymlinks(func(path string) error {
		files, err := file.ReadDirNames(path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if strings.HasPrefix(file, "temp") && strings.HasSuffix(file, "_input") {
				search[path] = true
				gotCoretemp = true
				break
			}
		}
		return nil
	})
	return
}

func HWMonSensors() ([]Sensor, error) {
	search := make(map[string]bool)
	gotCoretemp, err := hwmonSensors(search)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return nil, err
	}
	if !gotCoretemp {
		if gotCoretemp, err = coretempSensors(search); err != nil {
			return nil, err
		}
	}
	sensors := make([]Sensor, 0, len(search))
	for path := range search {
		name, err := file.SysRead(filepath.Join(path, "name"))
		if err != nil {
			continue
		}
		files, err := file.ReadDirNames(path)
		if err != nil {
			continue
		}
		for _, f := range files {
			fpath := filepath.Join(path, f)
			basepath, ok := strings.CutSuffix(fpath, "input")
			if !ok {
				continue
			}
			label, err := file.SysRead(basepath + "label")
			if err != nil {
				continue
			}
			max, _ := file.ReadInt(basepath + "max")
			if crit, _ := file.ReadInt(basepath + "crit"); crit > max {
				max = crit
			}
			sensors = append(sensors, Sensor{string(name), string(label), fpath, max, 0})
		}
	}
	return sensors, nil
}

func ThermalSensors() ([]Sensor, error) {
	d, err := Thermal()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return nil, err
	}
	defer d.Close()
	var sensors []Sensor
	err = d.WalkNames(func(name string) error {
		if !strings.HasPrefix(name, "thermal_zone") {
			return nil
		}
		p := filepath.Join(thermalClassPath, name)
		basepath, err := filepath.EvalSymlinks(p)
		if err != nil {
			basepath = p
		}
		path := filepath.Join(basepath, "temp")
		if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
			return nil
		}
		label, err := file.SysRead(filepath.Join(basepath, "type"))
		if err != nil {
			return nil
		}
		var max, crit int64
		for i := 0; true; i++ {
			fname := filepath.Join(basepath, "trip_point_"+strconv.Itoa(i)+"_temp")
			if _, err = os.Stat(fname); errors.Is(err, os.ErrNotExist) {
				break
			}
			typ, err := file.SysRead(filepath.Join(basepath, "trip_point_"+strconv.Itoa(i)+"_type"))
			if err != nil {
				continue
			}
			var val *int64
			switch string(typ) {
			case "high":
				val = &max
			case "critical":
				val = &crit
			default:
				continue
			}
			x, err := file.ReadInt(fname)
			if err != nil {
				continue
			}
			*val = x
		}
		if crit > max {
			max = crit
		}
		sensors = append(sensors, Sensor{name, string(label), path, max, 0})
		return nil
	})
	return sensors, err
}
