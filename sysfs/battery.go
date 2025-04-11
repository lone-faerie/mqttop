package sysfs

import (
	"io/fs"
	"log"
	"sync"
	"time"

	"github.com/lone-faerie/mqttop/internal/file"
)

type batteryFlag uint32

const (
	batteryCapacity batteryFlag = 1 << iota
	batteryCharge
	batteryEnergy
	batteryPower
	batteryCurrent
	batteryVoltage
	batteryStatus
	batteryTime
)

// Batt contains the paths to information for the battery.
type Batt struct {
	capacity    string
	chargeNow   string
	chargeFull  string
	energyNow   string
	energyFull  string
	powerNow    string
	currentNow  string
	voltageNow  string
	status      string
	timeToEmpty string

	isCharging bool
	flags      batteryFlag
	Kind       string
}

var (
	batteryDir  string
	batteryErr  error
	batteryOnce sync.Once
)

func getBattery() (string, error) {
	dirs, err := file.ReadDirPaths(powerSupplyPath)
	if err != nil {
		return "", err
	}

	for _, dir := range dirs {
		if !file.IsDir(dir) {
			continue
		}

		present, err := file.ReadInt(dir + file.Separator + "present")
		if err != nil || present != 1 {
			continue
		}

		typ, err := file.ReadString(dir + file.Separator + "type")
		if err == nil && (typ == "Battery" || typ == "UPS") {
			return dir, nil
		}
	}

	return "", fs.ErrNotExist
}

func findBattery() {
	batteryDir, batteryErr = getBattery()
}

// Battery returns the directory to the system's battery.
// If there is no battery on the system, Battery returns [fs.ErrNotExist]
func Battery() (*Dir, error) {
	batteryOnce.Do(findBattery)

	if batteryErr != nil {
		return nil, batteryErr
	}

	return file.OpenDir(batteryDir)
}

// GetBattery finds the system's battery and determines its supported
// features. If the is no battery on the system, GetBattery returns
// [fs.ErrNotExist]
func GetBattery() (*Batt, error) {
	dir, err := getBattery()
	if err != nil {
		return nil, err
	}

	var b Batt

	if path := dir + file.Separator + "capacity"; file.Exists(path) {
		b.capacity = path
		b.flags |= batteryCapacity
	}

	if path := dir + file.Separator + "charge_now"; file.Exists(path) {
		b.chargeNow = path
	}

	if path := dir + file.Separator + "charge_full"; file.Exists(path) {
		b.chargeFull = path
	}

	if path := dir + file.Separator + "energy_now"; file.Exists(path) {
		b.energyNow = path
	}

	if path := dir + file.Separator + "energy_full"; file.Exists(path) {
		b.energyFull = path
	}

	if path := dir + file.Separator + "power_now"; file.Exists(path) {
		b.powerNow = path
		b.flags |= batteryPower
	}

	if path := dir + file.Separator + "current_now"; file.Exists(path) {
		b.currentNow = path
		b.flags |= batteryCurrent
	}

	if path := dir + file.Separator + "voltage_now"; file.Exists(path) {
		b.voltageNow = path
		b.flags |= batteryVoltage
	}

	if path := dir + file.Separator + "status"; file.Exists(path) {
		b.status = path
		b.flags |= batteryStatus
	}

	if path := dir + file.Separator + "time_to_empty"; file.Exists(path) {
		b.timeToEmpty = path
		b.flags |= batteryTime
	}

	tech, err := file.ReadString(dir + file.Separator + "technology")
	if err == nil {
		b.Kind = tech
	}

	if b.chargeNow != "" && b.chargeFull != "" {
		b.flags |= batteryCharge
	}

	if b.energyNow != "" && b.energyFull != "" {
		b.flags |= batteryEnergy
	}

	return &b, nil
}

// ReadCapacity returns the contents of /sys/class/power_supply/<battery>/capacity.
func (b *Batt) ReadCapacity() (int64, error) {
	return file.ReadInt(b.capacity)
}

// ReadCharge returns the contents of /sys/class/power_supply/<battery>/charge_now and
// /sys/class/power_supply/<battery>/charge_full.
func (b *Batt) ReadCharge() (now, full int64, err error) {
	if now, err = file.ReadInt(b.chargeNow); err != nil {
		return
	}

	full, err = file.ReadInt(b.chargeFull)

	return
}

// ReadEnergy returns the contents of /sys/class/power_supply/<battery>/energy_now and
// /sys/class/power_supply/<battery>/energy_full.
func (b *Batt) ReadEnergy() (now, full int64, err error) {
	if now, err = file.ReadInt(b.energyNow); err != nil {
		return
	}

	full, err = file.ReadInt(b.energyFull)

	return
}

// ReadPower returns the contents of /sys/class/power_supply/<battery>/power_now.
func (b *Batt) ReadPower() (int64, error) {
	return file.ReadInt(b.powerNow)
}

// ReadCurrent returns the contents of /sys/class/power_supply/<battery>/current_now.
func (b *Batt) ReadCurrent() (int64, error) {
	return file.ReadInt(b.currentNow)
}

// ReadVoltage returns the contents of /sys/class/power_supply/<battery>/voltage_now.
func (b *Batt) ReadVoltage() (int64, error) {
	return file.ReadInt(b.voltageNow)
}

// ReadStatus returns the contents of /sys/class/power_supply/<battery>/status.
func (b *Batt) ReadStatus() (string, error) {
	return file.ReadLower(b.status)
}

// ReadTimeRemaining returns the contents of /sys/class/power_supply/<battery>/time_to_empty.
func (b *Batt) ReadTimeRemaining() (int64, error) {
	return file.ReadInt(b.timeToEmpty)
}

// HasCapacity returns true if b supports reading capacity.
func (b *Batt) HasCapacity() bool {
	return b.flags&batteryCapacity == batteryCapacity
}

// HasCharge returns true if b supports reading charge.
func (b *Batt) HasCharge() bool {
	return b.flags&batteryCharge == batteryCharge
}

// HasEnergy returns true if b supports reading energy.
func (b *Batt) HasEnergy() bool {
	return b.flags&batteryEnergy == batteryEnergy
}

// HasPower returns true if b supports reading power.
func (b *Batt) HasPower() bool {
	return b.flags&batteryPower == batteryPower
}

// HasCurrent returns true if b supports reading current.
func (b *Batt) HasCurrent() bool {
	return b.flags&batteryCurrent == batteryCurrent
}

// HasVoltage returns true if b supports reading voltage.
func (b *Batt) HasVoltage() bool {
	return b.flags&batteryVoltage == batteryVoltage
}

// HasTimeRemaining returns true if b supports reading time remaining.
func (b *Batt) HasTimeRemaining() bool {
	return b.flags&batteryTime == batteryTime
}

// Capacity returns the capacity of the battery. If b supports
// reading capacity, it is returned directly. Otherwise, the capacity
// is calculated from either charge or energy.
func (b *Batt) Capacity() (int, error) {
	var now, full string

	switch {
	case b.HasCapacity():
		i, err := file.ReadInt(b.capacity)

		return int(i), err
	case b.HasCharge():
		now = b.chargeNow
		full = b.chargeFull
	case b.HasEnergy():
		now = b.energyNow
		full = b.energyFull
	default:
		return 0, nil
	}

	n, err := file.ReadInt(now)
	if err != nil {
		return 0, err
	}

	f, err := file.ReadInt(full)
	if err != nil {
		return 0, err
	}

	if f == 0 {
		return -1, nil
	}

	return int(100 * n / f), nil
}

// Status returns the current status of b. One of "charging", "discharging", "not charging"
// "full", or "unknown".
func (b *Batt) Status() (string, error) {
	stat, err := file.ReadLower(b.status)
	if err == nil {
		b.isCharging = stat == "charging" || stat == "full"
	}

	return stat, err
}

// EstTimeRemaining returns the estimated time remaining of the battery life. This is calculated
// from either energy divied by power, charge divided by current, or the value of
// /sys/class/power_supply/<battery>/time_to_empty
func (b *Batt) EstTimeRemaining() (time.Duration, error) {
	const scale = uint64(time.Hour)

	var xp, yp string

	switch {
	case b.HasPower():
		log.Println("Using power")

		xp = b.energyNow
		yp = b.powerNow
	case b.HasCurrent():
		log.Println("Using current")

		xp = b.chargeNow
		yp = b.currentNow
	case b.HasTimeRemaining():
		log.Println("Using time_to_empty")

		x, err := file.ReadInt(b.timeToEmpty)

		return time.Duration(x), err
	default:
		log.Println("Unable to est time remaining")

		return 0, nil
	}

	x, err := file.ReadUint(xp)
	if err != nil {
		return 0, err
	}

	y, err := file.ReadUint(yp)
	if err != nil {
		return 0, err
	}

	if y == 0 {
		log.Println("Avoiding divide by zero")

		return -1, nil
	}

	return time.Duration(scale * x / y), nil
}
