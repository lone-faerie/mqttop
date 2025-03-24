package file

import (
	"os"
	"path/filepath"
	"slices"
	"unsafe"
)

const (
	dirNames uint8 = iota
	dirPaths
	dirSymlinks
)

type Dir struct {
	f         *os.File
	opened    bool
	names     []string
	namesType uint8
}

func OpenDir(name string) (*Dir, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	return &Dir{f: f, opened: true}, nil
}

func (d *Dir) Close() error {
	d.opened = false
	return d.f.Close()
}

func (d *Dir) Reset() error {
	if !d.opened {
		newF, err := open(d.f.Name())
		if err != nil {
			return err
		}
		d.f = newF
	}
	d.names = nil
	return nil
}

func (d *Dir) Name() string {
	return d.f.Name()
}

func (d *Dir) read() ([]os.FileInfo, error) {
	return d.f.Readdir(-1)
}

func (d *Dir) readNames(typ uint8) ([]string, error) {
	if len(d.names) == 0 || d.namesType != typ {
		names, err := d.f.Readdirnames(-1)
		if err != nil {
			return nil, err
		}
		d.names = names
		d.namesType = typ
	}
	return d.names, nil
}

func (d *Dir) ReadNames() ([]string, error) {
	return d.readNames(dirNames)
}

func dirPath(dirName, name string) string {
	return dirName + Separator + name
}

func (d *Dir) ReadPaths() ([]string, error) {
	names, err := d.readNames(dirPaths)
	if err != nil {
		return nil, err
	}
	dirName := d.Name()
	for i, name := range names {
		names[i] = dirPath(dirName, name)
	}
	return names, nil
}

func dirSymlink(dirName, name string) string {
	name = dirPath(dirName, name)
	path, err := filepath.EvalSymlinks(name)
	if err != nil {
		return name
	}
	return path
}

func (d *Dir) ReadSymlinks() ([]string, error) {
	names, err := d.readNames(dirSymlinks)
	if err != nil {
		return nil, err
	}
	dirName := d.Name()
	for i, name := range names {
		names[i] = dirSymlink(dirName, name)
	}
	return names, nil
}

func (d *Dir) WalkNames(fn func(string) error) error {
	names, err := d.readNames(dirNames)
	if err != nil {
		return err
	}
	for i := range names {
		if err = fn(names[i]); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dir) WalkPaths(fn func(string) error) error {
	names, err := d.ReadNames()
	if err != nil {
		return err
	}
	dirName := d.Name()
	for _, name := range names {
		name = dirPath(dirName, name)
		if err = fn(name); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dir) WalkSymlinks(fn func(string) error) error {
	names, err := d.ReadNames()
	if err != nil {
		return err
	}
	dirName := d.Name()
	for _, name := range names {
		name = dirSymlink(dirName, name)
		if err = fn(name); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dir) Walk(fn func(string, bool) error) error {
	names, err := d.ReadNames()
	if err != nil {
		return err
	}
	dirName := d.Name()
	for _, name := range names {
		name = dirPath(dirName, name)
		info, err := Stat(name)
		if err != nil {
			return err
		}
		if err = fn(name, info.IsDir()); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dir) Contains(name string) bool {
	names, err := d.ReadNames()
	if err != nil {
		return false
	}
	for i := range names {
		if names[i] == name {
			return true
		}
	}
	return false
}

func (d *Dir) ContainsFunc(fn func(string) bool) bool {
	names, err := d.ReadNames()
	if err != nil {
		return false
	}
	for i := range names {
		if fn(names[i]) {
			return true
		}
	}
	return false
}

func (d *Dir) ContainsAll(name ...string) bool {
	names, err := d.ReadNames()
	if err != nil {
		return false
	}
	if len(names) < len(name) {
		return false
	}
	for i := range name {
		if !slices.Contains(names, name[i]) {
			return false
		}
	}
	return true
}

func (d *Dir) ContainsAllFunc(fn func(string) bool) bool {
	names, err := d.ReadNames()
	if err != nil {
		return false
	}
	for i := range names {
		if !fn(names[i]) {
			return false
		}
	}
	return true
}

func (d *Dir) Open(name string) (*File, error) {
	return Open(filepath.Join(d.f.Name(), name))
}

func (d *Dir) OpenDir(name string) (*Dir, error) {
	return OpenDir(filepath.Join(d.f.Name(), name))
}

func (d *Dir) Read(name string) ([]byte, error) {
	return Read(filepath.Join(d.f.Name(), name))
}

func (d *Dir) ReadBytes(name string) ([]byte, error) {
	return SysRead(filepath.Join(d.f.Name(), name))
}

func (d *Dir) ReadString(name string) (string, error) {
	b, err := d.ReadBytes(name)
	return unsafe.String(unsafe.SliceData(b), len(b)), err
}

func (d *Dir) ReadUint(name string) (uint64, error) {
	return ReadUint(filepath.Join(d.f.Name(), name))
}

func (d *Dir) ReadInt(name string) (int64, error) {
	return ReadInt(filepath.Join(d.f.Name(), name))
}
