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

// Dir wraps an [os.File] for conveniently working with files
// within the directory.
type Dir struct {
	f         *os.File
	opened    bool
	names     []string
	namesType uint8
}

// Open opens the named directory. If successful, methods on the returned
// directory can be used for reading; the associated file descriptor has mode
// O_RDONLY. The path of the directory will be prefixed with the root path,
// either / or the value of $MQTTOP_ROOTFS_PATH.
func OpenDir(name string) (*Dir, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}

	return &Dir{f: f, opened: true}, nil
}

// Close closes the underlying [os.File] of d, rendering it unusable for I/O.
func (d *Dir) Close() error {
	d.opened = false

	return d.f.Close()
}

// Reset prepares the directort to be read again from the beginning. If
// the directory is currently open, this is a no-op, otherwise the directory
// will be reopened.
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

// Name returns the name of the file as presented to Open.
//
// It is safe to call Name after [Dir.Close].
func (d *Dir) Name() string {
	return d.f.Name()
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

// ReadNames reads the contents of the directory and returns a slice of the names of files in the directory.
func (d *Dir) ReadNames() ([]string, error) {
	return d.readNames(dirNames)
}

func dirPath(dirName, name string) string {
	return dirName + Separator + name
}

// ReadPaths reads the contents of the directory and returns a slice of the paths of files in the directory.
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

// ReadSymlinks reads the contents of the directory and returns a slice of the paths of files in the directory
// after following symlinks.
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

// WalkNames reads the contents of the directory and performs fn on each name of files in the directory.
// If fn returns a non-nil error, WalkNames stops and returns the error.
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

// WalkPaths reads the contents of the directory and performs fn on each path of files in the directory.
// If fn returns a non-nil error, WalkPaths stops and returns the error.
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

// WalkSymlinks reads the contents of the directory and performs fn on each path of files in the directory
// after following symlinks. If fn returns a non-nil error, WalkSymlinks stops and returns the error.
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

// Walk reads the contents of the directory and performs fn on each file in the directory, passing the
// file name and if the file is a directory. If fn returns a non-nil error, WalkNames stops and returns
// the error.
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

// Contains reports whether the directory contains a file named name.
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

// Contains reports whether the directory contains a file with a name that satisfies fn(name).
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

// ContainsAll reports whether the directory contains files with all the given names.
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

// ContainsAll reports whether the directory contains files with names that all satisfy fn(name).
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

// Open opens the named file in dir for reading.
func (d *Dir) Open(name string) (*File, error) {
	return Open(filepath.Join(d.f.Name(), name))
}

// OpenDir opens the named directory in dir for reading.
func (d *Dir) OpenDir(name string) (*Dir, error) {
	return OpenDir(filepath.Join(d.f.Name(), name))
}

// Read reads the named file in dir and returns the contents.
func (d *Dir) Read(name string) ([]byte, error) {
	return Read(filepath.Join(d.f.Name(), name))
}

// ReadBytes reads the named file in dir using syscalls and returns the contents.
func (d *Dir) ReadBytes(name string) ([]byte, error) {
	return SysRead(filepath.Join(d.f.Name(), name))
}

// ReadString reads the named file in dir using syscalls and returns the contents as a string.
func (d *Dir) ReadString(name string) (string, error) {
	b, err := d.ReadBytes(name)

	return unsafe.String(unsafe.SliceData(b), len(b)), err
}

// ReadUint reads the named file in dir using syscalls and returns the contents parsed as a uint64.
func (d *Dir) ReadUint(name string) (uint64, error) {
	return ReadUint(filepath.Join(d.f.Name(), name))
}

// ReadUint reads the named file in dir using syscalls and returns the contents parsed as a int64.
func (d *Dir) ReadInt(name string) (int64, error) {
	return ReadInt(filepath.Join(d.f.Name(), name))
}
