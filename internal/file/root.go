package file

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	root string = "/"
)

func init() {
	if s, ok := os.LookupEnv("MQTTOP_ROOTFS_PATH"); ok && len(s) > 0 {
		root = s
	}
}

func abs(name string) (string, error) {
	if strings.HasPrefix(name, root) {
		return name, nil
	}

	name, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	if root == "/" {
		return name, nil
	}

	return filepath.Join(root, name[1:]), nil
}

func open(name string) (*os.File, error) {
	name, err := abs(name)
	if err != nil {
		return nil, err
	}

	return os.Open(name)
}

func sysOpen(name string) (int, error) {
	name, err := abs(name)
	if err != nil {
		return 0, err
	}

	return unix.Open(name, unix.O_RDONLY, 0)
}
