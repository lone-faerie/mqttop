package file

import (
	"io"
	"os"
	//	"log"

	"github.com/lone-faerie/mqttop/log"
)

func Read(name string) ([]byte, error) {
	const maxBufferSize = 1024 * 1024
	f, err := open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func ReadDir(name string) ([]os.DirEntry, error) {
	name, err := abs(name)
	if err != nil {
		return nil, err
	}
	return os.ReadDir(name)
}

func ReadDirNames(name string) ([]string, error) {
	log.Debug("ReadDirNames", "name", name)
	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.ReadNames()
}

func ReadDirPaths(name string) ([]string, error) {
	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.ReadPaths()
}

func ReadDirSymlinks(name string) ([]string, error) {
	d, err := OpenDir(name)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.ReadSymlinks()
}
