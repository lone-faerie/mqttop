package procfs

import (
	"strconv"

	"github.com/lone-faerie/mqttop/internal/file"
)

type Proc struct {
	pid int
	dir string
}

func Procs() ([]Proc, error) {
	d, err := file.OpenDir(MountPath)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	names, err := d.ReadNames()
	if err != nil {
		return nil, err
	}
	procs := make([]Proc, 0, len(names))
	for _, name := range names {
		pid, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		procs = append(procs, Proc{pid, Path(name)})
	}
	return procs, nil
}
