package sysfs

import (
	"errors"
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
)

type Vendor uint32

const (
	MSI    Vendor = 0x1462
	Nvidia Vendor = 0x10de
)

func GPUVendor() (Vendor, error) {
	devs, err := file.ReadDirPaths(pciDevicesPath)
	if err != nil {
		return 0, err
	}

	for _, dev := range devs {
		b, err := file.ReadBytes(filepath.Join(dev, "class"))
		if err != nil {
			continue
		}

		class := byteutil.Btox(b)
		if class&0x0300 != 0x0300 {
			continue
		}

		b, err = file.ReadBytes(filepath.Join(dev, "vendor"))
		if err != nil {
			continue
		}

		vendor := Vendor(byteutil.Btox(b))
		if vendor == Nvidia || vendor == MSI {
			return vendor, nil
		}
	}

	return 0, errors.New("GPU not found")
}
