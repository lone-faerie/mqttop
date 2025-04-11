package file

import (
	"io"
	"os"

	"github.com/lone-faerie/mqttop/log"
)

// Read reads the named file and returns the contents.
func Read(name string) ([]byte, error) {
	f, err := open(name)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	return io.ReadAll(f)
}

// ReadNames reads the contents of the named directory and returns all its directory entries.
func ReadDir(name string) ([]os.DirEntry, error) {
	name, err := abs(name)
	if err != nil {
		return nil, err
	}

	return os.ReadDir(name)
}

// ReadDirNames reads the contents of the named directory and returns a slice of the names of files
// in the directory.
func ReadDirNames(name string) ([]string, error) {
	log.Debug("ReadDirNames", "name", name)

	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}

	defer d.Close()

	return d.ReadNames()
}

// ReadDirPaths reads the contents of the named directory and returns a slice of the paths of files
// in the directory.
func ReadDirPaths(name string) ([]string, error) {
	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}

	defer d.Close()

	return d.ReadPaths()
}

// ReadDirSymlinks reads the contents of the named directory and returns a slice of the paths of files
// in the directory after following symlinks.
func ReadDirSymlinks(name string) ([]string, error) {
	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}

	defer d.Close()

	return d.ReadSymlinks()
}
