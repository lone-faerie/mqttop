package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/sysfs"
	"github.com/lone-faerie/mqttop/log"
)

type batteryUpdates struct {
	capacity      bool
	charge        bool
	energy        bool
	power         bool
	current       bool
	voltage       bool
	timeRemaining bool
}

type batteryFlag uint16

const (
	batteryCapacity batteryFlag = 1 << iota
	batteryCharge
	batteryEnergy
	batteryPower
	batteryCurrent
	batteryVoltage
	batteryTime
	batteryStatus
)

func (f batteryFlag) Has(flags batteryFlag) bool {
	return f&flags != 0
}

func (f batteryFlag) String() string {
	var s []string
	if f.Has(batteryCapacity) {
		s = append(s, "capacity")
	}
	if f.Has(batteryCharge) {
		s = append(s, "charge")
	}
	if f.Has(batteryEnergy) {
		s = append(s, "energy")
	}
	if f.Has(batteryPower) {
		s = append(s, "power")
	}
	if f.Has(batteryCurrent) {
		s = append(s, "current")
	}
	if f.Has(batteryVoltage) {
		s = append(s, "voltage")
	}
	if f.Has(batteryTime) {
		s = append(s, "time")
	}
	return fmt.Sprintf("%s (%08b)", strings.Join(s, "|"), f)
}

type Battery struct {
	bat *sysfs.Batt

	kind          string
	capacity      int
	chargeNow     int64
	chargeFull    int64
	energyNow     int64
	energyFull    int64
	power         int64
	current       int64
	voltage       int64
	status        string
	timeRemaining time.Duration

	flags   batteryFlag
	updates batteryFlag
	changes batteryFlag

	interval time.Duration
	tick     *time.Ticker
	topic    string

	mu   sync.RWMutex
	once sync.Once
	stop context.CancelFunc
	ch   chan error
}

func NewBattery(cfg *config.Config) (*Battery, error) {
	b := &Battery{}
	bat, err := sysfs.GetBattery()
	if err != nil {
		return nil, errNotSupported(b.Type(), err)
	}
	b.bat = bat
	b.setFlags()
	if cfg.Battery.Interval > 0 {
		b.interval = cfg.Battery.Interval
	} else {
		b.interval = cfg.Interval
	}
	if cfg.Battery.Topic != "" {
		b.topic = cfg.Battery.Topic
	} else {
		b.topic = "mqtt/metric/battery"
	}
	return b, nil
}

func (b *Battery) has(flag batteryFlag) bool {
	return b.flags.Has(flag)
}

func (b *Battery) hasCapacity() bool {
	const flags = batteryCapacity | batteryCharge | batteryEnergy
	return b.flags.Has(flags)
}

func (b *Battery) hasTimeRemaining() bool {
	const (
		energyPower   = batteryEnergy | batteryPower
		chargeCurrent = batteryCharge | batteryCurrent
	)
	return b.flags.Has(energyPower) || b.flags.Has(chargeCurrent) || b.flags.Has(batteryTime)
}

func (b *Battery) setFlag(hasFlag func() bool, flag batteryFlag) {
	if hasFlag() {
		b.flags |= flag
	}
}

func (b *Battery) setFlags() {
	b.setFlag(b.bat.HasCapacity, batteryCapacity)
	b.setFlag(b.bat.HasCharge, batteryCharge)
	b.setFlag(b.bat.HasEnergy, batteryEnergy)
	b.setFlag(b.bat.HasPower, batteryPower)
	b.setFlag(b.bat.HasCurrent, batteryCurrent)
	b.setFlag(b.bat.HasVoltage, batteryVoltage)
	b.setFlag(b.bat.HasTimeRemaining, batteryTime)
}

func (*Battery) Type() string {
	return "battery"
}

func (b *Battery) Topic() string {
	return b.topic
}

func (b *Battery) SetInterval(d time.Duration) {
	b.mu.Lock()
	if b.tick != nil && d != b.interval {
		b.tick.Reset(d)
	}
	b.interval = d
	b.mu.Unlock()
}

func (b *Battery) loop(ctx context.Context) {
	b.mu.Lock()
	b.tick = time.NewTicker(b.interval)
	b.mu.Unlock()

	defer b.tick.Stop()
	defer close(b.ch)
	var (
		err error
		ch  chan error
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.tick.C:
			err = b.Update()
			if err == ErrNoChange {
				log.Debug("battery updated, no change")
				break
			}
			log.Debug("battery updated")
			ch = b.ch
		case ch <- err:
			ch = nil
		}
	}
}

func (b *Battery) Start(ctx context.Context) (err error) {
	if b.interval == 0 {
		return
	}
	b.once.Do(func() {
		ctx, b.stop = context.WithCancel(ctx)
		b.ch = make(chan error)
		go b.loop(ctx)
	})
	return
}

func (b *Battery) updateCapacity() error {
	var now, full int64
	var err error
	switch {
	case b.flags.Has(batteryCapacity):
		if now, err = b.bat.ReadCapacity(); err != nil {
			return err
		}
		if int(now) != b.capacity {
			b.changes |= batteryCapacity
		}
		b.capacity = int(now)
		b.updates |= batteryCapacity
		return nil
	case b.flags.Has(batteryCharge):
		if err = b.updateCharge(); err != nil {
			return err
		}
		now = b.chargeNow
		full = b.chargeFull
	case b.flags.Has(batteryEnergy):
		if err = b.updateEnergy(); err != nil {
			return err
		}
		now = b.energyNow
		full = b.energyFull
	default:
		return nil
	}
	b.capacity = int(100 * now / full)
	return nil
}

func (b *Battery) updateCharge() error {
	if b.updates.Has(batteryCharge) {
		return nil
	}
	now, full, err := b.bat.ReadCharge()
	if err != nil {
		return err
	}
	if now != b.chargeNow && full != b.chargeFull {
		b.changes |= batteryCharge
	}
	b.chargeNow = now
	b.chargeFull = full
	b.updates |= batteryCharge
	return nil
}

func (b *Battery) updateEnergy() error {
	if b.updates.Has(batteryEnergy) {
		return nil
	}
	now, full, err := b.bat.ReadEnergy()
	if err != nil {
		return err
	}
	if now != b.energyNow && full != b.energyFull {
		b.changes |= batteryEnergy
	}
	b.energyNow = now
	b.energyFull = full
	b.updates |= batteryEnergy
	return nil
}

func (b *Battery) updatePower() error {
	if b.updates.Has(batteryPower) {
		return nil
	}
	p, err := b.bat.ReadPower()
	if err != nil {
		return err
	}
	if p != b.power {
		b.changes |= batteryPower
	}
	b.power = p
	b.updates |= batteryPower
	return nil
}

func (b *Battery) updateCurrent() error {
	if b.updates.Has(batteryCurrent) {
		return nil
	}
	c, err := b.bat.ReadCurrent()
	if err != nil {
		return err
	}
	if c != b.current {
		b.changes |= batteryCurrent
	}
	b.current = c
	b.updates |= batteryCurrent
	return nil
}

func (b *Battery) updateVoltage() error {
	if b.updates.Has(batteryVoltage) {
		return nil
	}
	v, err := b.bat.ReadVoltage()
	if err != nil {
		return err
	}
	if v != b.voltage {
		b.changes |= batteryVoltage
	}
	b.voltage = v
	b.updates |= batteryVoltage
	return nil
}

func (b *Battery) updateTimeRemaining() error {
	const scale = uint64(time.Hour)
	var x, y uint64
	switch {
	case b.flags.Has(batteryEnergy | batteryPower):
		if err := b.updateEnergy(); err != nil {
			return err
		}
		if err := b.updatePower(); err != nil {
			return err
		}
		x = uint64(b.energyNow)
		y = uint64(b.power)
	case b.flags.Has(batteryCharge | batteryCurrent):
		if err := b.updateCharge(); err != nil {
			return err
		}
		if err := b.updateCurrent(); err != nil {
			return err
		}
		x = uint64(b.chargeNow)
		y = uint64(b.current)
	case b.flags.Has(batteryTime):
		t, err := b.bat.ReadTimeRemaining()
		if err != nil {
			return err
		}
		rem := time.Duration(t) * time.Minute
		if rem != b.timeRemaining {
			b.changes |= batteryTime
		}
		b.timeRemaining = rem
		b.updates |= batteryTime
	}
	if y == 0 {
		b.timeRemaining = -1
		return nil
	}
	b.timeRemaining = time.Duration(scale * x / y)
	return nil
}

func (b *Battery) Update() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.updates = 0
	b.changes = 0
	if err := b.updateCapacity(); err != nil {
		return err
	}
	s, err := b.bat.ReadStatus()
	if err != nil {
		return err
	}
	if s != b.status {
		b.changes |= batteryStatus
	}
	b.status = s
	if s != "charging" && s != "full" {
		if err := b.updateTimeRemaining(); err != nil {
			return err
		}
	}
	switch {
	case b.flags.Has(batteryPower):
		if err := b.updatePower(); err != nil {
			return err
		}
	case b.flags.Has(batteryCurrent | batteryVoltage):
		if err := b.updateCurrent(); err != nil {
			return err
		}
		if err := b.updateVoltage(); err != nil {
			return err
		}
		if b.voltage == 0 {
			b.power = -1
		} else if b.current == 0 {
			b.power = 0
		} else {
			b.power = (b.current / 1000) * (b.voltage / 1000)
		}
	}
	if b.changes == 0 {
		return ErrNoChange
	}
	return nil
}

func (b *Battery) Updated() <-chan error {
	return b.ch
}

func (b *Battery) Stop() {
	b.mu.Lock()
	if b.stop != nil {
		b.stop()
	}
	b.mu.Unlock()
}

func (bat *Battery) String() string {
	bat.mu.RLock()
	defer bat.mu.RUnlock()
	var b strings.Builder
	b.Write([]byte{' ', ' '})
	b.WriteString(bat.bat.Kind)
	return b.String()
}

func (bat *Battery) AppendText(b []byte) ([]byte, error) {
	bat.mu.RLock()
	defer bat.mu.RUnlock()
	b = append(b, "{\"kind\": \""...)
	b = append(b, bat.bat.Kind...)
	b = append(b, "\", \"status\": \""...)
	b = append(b, bat.status...)
	b = append(b, '"')
	if bat.hasCapacity() {
		b = append(b, ", \"capacity\": "...)
		b = strconv.AppendInt(b, int64(bat.capacity), 10)
	}
	if bat.flags.Has(batteryPower) {
		b = append(b, ", \"power\": "...)
		b = byteutil.AppendDecimal(b, bat.power, 6)
	}
	if bat.hasTimeRemaining() && bat.timeRemaining > 0 {
		b = append(b, ", \"timeRemaining\": "...)
		b = strconv.AppendInt(b, int64(bat.timeRemaining/time.Second), 10)
	}
	return append(b, '}'), nil
}

func (bat *Battery) MarshalJSON() ([]byte, error) {
	return bat.AppendText(nil)
}
