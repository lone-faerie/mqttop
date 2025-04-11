// Package procfs provides access to various files in /sys and /etc
package sysfs

import (
	"path/filepath"

	"github.com/lone-faerie/mqttop/internal/file"
)

const MountPath = file.Separator + "sys" // /sys

const (
	classPath        = MountPath + file.Separator + "class"                       // /sys/class
	hwmonClassPath   = classPath + file.Separator + "hwmon"                       // /sys/class/hwmon
	thermalClassPath = classPath + file.Separator + "thermal"                     // /sys/class/thermal
	netClassPath     = classPath + file.Separator + "net"                         // /sys/class/net
	powerSupplyPath  = classPath + file.Separator + "power_supply"                // /sys/class/power_supply
	dmiClassPath     = classPath + file.Separator + "dmi"                         // /sys/class/dmi
	dmiIDPath        = classPath + file.Separator + "dmi" + file.Separator + "id" // /sys/class/dmi/id
)

const (
	devicesPath         = MountPath + file.Separator + "devices"                                         // /sys/devices
	platformDevicesPath = devicesPath + file.Separator + "platform"                                      // /sys/devices/platform
	systemDevicesPath   = devicesPath + file.Separator + "system"                                        // /sys/devices/system
	coretempPath        = platformDevicesPath + file.Separator + "coretemp.0" + file.Separator + "hwmon" // /sys/devices/platfotm/coretemp.0/hwmon
	cpuDevicesPath      = systemDevicesPath + file.Separator + "cpu"                                     // /sys/devices/system/cpu
)

const (
	busPath        = MountPath + file.Separator + "bus"      // /sys/bus
	pciBusPath     = busPath + file.Separator + "pci"        // /sys/bus/pci
	pciDevicesPath = pciBusPath + file.Separator + "devices" // /sys/bus/pci/devices
)

type (
	File = file.File
	Dir  = file.Dir
)

func Path(elem ...string) string {
	return MountPath + file.Separator + filepath.Join(elem...)
}

// HWMon returns the directory /sys/class/hwmon
func HWMon() (*Dir, error) {
	return file.OpenDir(hwmonClassPath)
}

// Thermal returns the directory /sys/class/thermal
func Thermal() (*Dir, error) {
	return file.OpenDir(thermalClassPath)
}

// Coretemp returns the directory /sys/devices/platform/coretemp.0/hwmon
func Coretemp() (*Dir, error) {
	return file.OpenDir(coretempPath)
}

// CPU returns the directory /sys/devices/system/cpu
func CPU() (*Dir, error) {
	return file.OpenDir(cpuDevicesPath)
}

// NetDevice returns the directory /sys/class/net/<iface>
func NetDevice(iface string) (*Dir, error) {
	return file.OpenDir(netClassPath + file.Separator + iface)
}

// NetStatistics returns the contents of /sys/class/net/<iface>/statistics/rx_bytes and
// /sys/class/net/<iface>/statistics/tx_bytes
func NetStatistics(iface string) (rx, tx uint64, err error) {
	path := netClassPath + file.Separator + iface + file.Separator + "statistics"
	if rx, err = file.ReadUint(path + file.Separator + "rx_bytes"); err != nil {
		return
	}

	tx, err = file.ReadUint(path + file.Separator + "tx_bytes")

	return
}

// PowerSupply returns the directory /sys/class/power_supply
func PowerSupply() (*Dir, error) {
	return file.OpenDir(powerSupplyPath)
}

// DMI returns the directory /sys/class/dmi
func DMI() (*Dir, error) {
	return file.OpenDir(dmiIDPath)
}
