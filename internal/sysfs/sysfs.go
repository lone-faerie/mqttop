package sysfs

import (
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/file"
)

const MountPath = file.Separator + "sys"

const (
	classPath        = MountPath + file.Separator + "class"
	hwmonClassPath   = classPath + file.Separator + "hwmon"
	thermalClassPath = classPath + file.Separator + "thermal"
	netClassPath     = classPath + file.Separator + "net"
	powerSupplyPath  = classPath + file.Separator + "power_supply"
	dmiClassPath     = classPath + file.Separator + "dmi"
	dmiIDPath        = classPath + file.Separator + "dmi" + file.Separator + "id"
)

const (
	devicesPath         = MountPath + file.Separator + "devices"
	platformDevicesPath = devicesPath + file.Separator + "platform"
	systemDevicesPath   = devicesPath + file.Separator + "system"
	coretempPath        = platformDevicesPath + file.Separator + "coretemp.0" + file.Separator + "hwmon"
	cpuDevicesPath      = systemDevicesPath + file.Separator + "cpu"
)

const (
	busPath        = MountPath + file.Separator + "bus"
	pciBusPath     = busPath + file.Separator + "pci"
	pciDevicesPath = pciBusPath + file.Separator + "devices"
)

type (
	File = file.File
	Dir  = file.Dir
)

func Path(elem ...string) string {
	return MountPath + file.Separator + filepath.Join(elem...)
}

func HWMon() (*Dir, error) {
	return file.OpenDir(hwmonClassPath)
}

func Thermal() (*Dir, error) {
	return file.OpenDir(thermalClassPath)
}

func Coretemp() (*Dir, error) {
	return file.OpenDir(coretempPath)
}

func CPU() (*Dir, error) {
	return file.OpenDir(cpuDevicesPath)
}

func NetDevice(iface string) (*Dir, error) {
	return file.OpenDir(netClassPath + file.Separator + iface)
}

func NetStatistics(iface string) (rx, tx uint64, err error) {
	path := netClassPath + file.Separator + iface + file.Separator + "statistics"
	if rx, err = file.ReadUint(path + file.Separator + "rx_bytes"); err != nil {
		return
	}
	tx, err = file.ReadUint(path + file.Separator + "tx_bytes")
	return
}

func PowerSupply() (*Dir, error) {
	return file.OpenDir(powerSupplyPath)
}

func DMI() (*Dir, error) {
	return file.OpenDir(dmiIDPath)
}
