package file

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/lone-faerie/mqttop/log"
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
	name, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	if root == "/" {
		return name, nil
	}

	if strings.HasPrefix(name, root) {
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

// SetRoot sets the root directory to open files in. Only used during testing.
//
// If the environment variable $MQTTOP_ROOTFS_PATH is set, this is automatically
// handled on init.
func SetRoot(s string) error {
	s, err := filepath.EvalSymlinks(s)
	if err != nil {
		return err
	}

	s, err = filepath.Abs(s)
	if err != nil {
		return err
	}
	log.Debug("Setting root", "path", s)
	root = s
	return nil
}
