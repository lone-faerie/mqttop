package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/procfs"
	"github.com/lone-faerie/mqttop/sysfs"
)

type cpuCore struct {
	logical  int
	physical int
	baseFreq int64
	currFreq int64
	minFreq  int64
	maxFreq  int64
	freq     sysfs.CPUFreq
	temp     *sysfs.Sensor
	total    uint64
	idle     uint64
	percent  int
}

var (
	coreCount = runtime.NumCPU()
)

type cpuFlag byte

const (
	cpuTemperature cpuFlag = 1 << iota
	cpuFrequency
	cpuUsage
)

func (f cpuFlag) Has(flags cpuFlag) bool {
	return f&flags != 0
}

// CPU implements the [Metric] interface to provide the system processor
// metrics. This includes the usage, frequency, and temperature of the CPU
// and each of its cores.
type CPU struct {
	Name    string
	cores   []cpuCore
	temps   []sysfs.Sensor
	temp    *sysfs.Sensor
	coremap []int

	total   uint64
	idle    uint64
	percent int

	flags cpuFlag

	interval time.Duration
	tick     *time.Ticker
	topic    string

	selectFn   func() (temp, freq int64)
	selectMode string
	rand       *rand.Rand

	mu   sync.RWMutex
	once sync.Once
	stop context.CancelFunc
	ch   chan error
}

// NewCPU returns a new [CPU] initialized from cfg. If there is any error
// encountered while initializing the CPU, a non-nil error that wraps [ErrNotSupported]
// is returned.
func NewCPU(cfg *config.Config) (*CPU, error) {
	c := &CPU{
		Name:  cfg.CPU.Name,
		cores: make([]cpuCore, coreCount),
	}

	if err := c.init(); err != nil {
		return nil, errNotSupported(c.Type(), err)
	}

	c.setSelectionMode(cfg.CPU.SelectionMode)
	if c.selectFn == nil {
		c.selectMode = "auto"
		c.selectFn = c.SelectAuto
	}

	if cfg.CPU.Interval > 0 {
		c.interval = cfg.CPU.Interval
	} else {
		c.interval = cfg.Interval
	}

	if cfg.CPU.Topic != "" {
		c.topic = cfg.CPU.Topic
	} else if cfg.BaseTopic != "" {
		c.topic = cfg.BaseTopic + "/metric/cpu"
	} else {
		c.topic = "mqttop/metric/cpu"
	}

	c.Name = cfg.CPU.FormatName(c.Name)

	return c, nil
}

func (c *CPU) init() (err error) {
	if err = c.parseInfo(); err != nil {
		return
	}

	if err = c.findSensors(); err == nil {
		c.flags |= cpuTemperature
	}

	if err = c.findFreqs(); err == nil {
		c.flags |= cpuFrequency
	}

	c.flags |= cpuUsage

	return nil
}

func (c *CPU) parseInfo() error {
	info, err := procfs.CPUInfo()
	if err != nil {
		return err
	}

	log.Debug("parseInfo", "Opened", info.Name())
	defer info.Close()

	var (
		logical  int
		physical int
	)

	for {
		line, err := info.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if len(line) == 0 {
			if n := logical + 1; n > len(c.cores) {
				c.cores = slices.Grow(c.cores, n-len(c.cores))
			}

			core := &c.cores[logical]
			core.logical = logical
			core.physical = physical
		}

		key, val := byteutil.Field(line)

		switch string(key) {
		case "processor":
			logical = int(byteutil.Btou(val))
		case "model name":
			if len(c.Name) == 0 {
				c.Name = string(bytes.TrimSpace(val))
			}
		case "core id":
			physical = int(byteutil.Btou(val))
		}
	}

	slices.SortFunc(c.cores, func(a, b cpuCore) int {
		return a.logical - b.logical
	})

	c.coremap = make([]int, len(c.cores))

	for i := range c.cores {
		c.coremap[i] = c.cores[i].physical
	}

	return nil
}

func (c *CPU) findSensors() error {
	sensors, err := sysfs.HWMonSensors()
	if err != nil {
		return err
	}

	var coreSensors []sysfs.Sensor

	for i := range sensors {
		label := sensors[i].Label
		if strings.HasPrefix(label, "Package id") || strings.HasPrefix(label, "Tdie") {
			if c.temp == nil {
				c.temp = new(sysfs.Sensor)
			}

			*c.temp = sensors[i]
		} else if strings.Contains(label, "Core") || strings.HasPrefix(label, "Tccd") {
			coreSensors = append(coreSensors, sensors[i])
		}
	}

	if c.temp == nil {
		log.Debug("No hwmon sensors found")
		sensors, err = sysfs.ThermalSensors()
		if err != nil {
			return err
		}

		for i := range sensors {
			label := strings.ToLower(sensors[i].Label)
			if strings.Contains(label, "core") || strings.Contains(label, "k10temp") {
				c.temp = new(sysfs.Sensor)
				*c.temp = sensors[i]

				break
			}
		}
	}
	if c.temp == nil {
		log.Debug("No thermal sensors found")
	}

	slices.SortFunc(coreSensors, func(a, b sysfs.Sensor) int {
		return strings.Compare(a.Label, b.Label)
	})

	c.temps = slices.Clip(coreSensors)

	for i := range c.temps {
		idx := i

		if istr, ok := strings.CutPrefix(c.temps[i].Label, "Core "); ok {
			if x, err := strconv.Atoi(istr); err == nil {
				idx = x
			}
		}

		for j := range c.cores {
			if c.cores[j].physical == idx && c.cores[j].temp == nil {
				c.cores[j].temp = &c.temps[i]
			}
		}
	}

	return nil
}

func (c *CPU) findFreqs() error {
	freqs, err := sysfs.CPUFreqs()
	if err != nil {
		return err
	}

	log.Debug("findFreqs", "freqs", len(freqs))

	for i := range c.cores {
		if i >= len(freqs) {
			break
		}

		c.cores[i].freq = freqs[i]
	}

	return nil
}

// Type returns the metric type, "cpu".
func (c *CPU) Type() string {
	return "cpu"
}

// Topic returns the topic to publish cpu metrics to.
func (c *CPU) Topic() string {
	return c.topic
}

// SetInterval sets the update interval for the metric.
func (c *CPU) SetInterval(d time.Duration) {
	if d == 0 {
		c.Stop()
		return
	}

	c.mu.Lock()

	if c.tick != nil && d != c.interval {
		c.tick.Reset(d)
	}

	c.interval = d

	c.mu.Unlock()
}

func (c *CPU) loop(ctx context.Context) {
	c.mu.Lock()
	c.tick = time.NewTicker(c.interval)
	c.mu.Unlock()

	defer c.tick.Stop()
	defer close(c.ch)

	var (
		err error
		ch  chan error
	)

	log.Debug("cpu started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.tick.C:
			err = c.Update()
			if err == ErrNoChange {
				log.Debug("cpu updated, no change")
				break
			} else {
				log.Debug("cpu updated")
			}

			ch = c.ch
		case ch <- err:
			ch = nil
		}
	}
}

// Start starts the cpu updating. If ctx is cancelled or
// times out, the metric will stop and may not be restarted.
func (c *CPU) Start(ctx context.Context) (err error) {
	if c.interval == 0 {
		log.Warn("CPU interval is 0, not starting")
		return
	}

	c.once.Do(func() {
		ctx, c.stop = context.WithCancel(ctx)
		c.ch = make(chan error)

		go c.loop(ctx)
	})

	return
}

func (c *CPU) updateUsage() error {
	stat, err := procfs.Stat()
	if err != nil {
		return err
	}

	defer stat.Close()

	var (
		name   []byte
		buf    []byte
		cpuNum int
	)

	for {
		line, err := stat.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if len(line) == 0 {
			continue
		}

		if line[0] != 'c' {
			break
		}

		name, line = byteutil.Column(line)

		if len(name) > 3 {
			cpuNum = int(byteutil.Btoi(name[3:]))
		} else {
			cpuNum = -1
		}

		var (
			times         [8]uint64
			val           uint64
			total, idle   uint64
			dTotal, dIdle uint64
		)

		for i := 0; len(line) > 0 && i < len(times); i++ {
			buf, line = byteutil.Column(line)
			val = byteutil.Btou(buf)
			total += val
			times[i] = val
		}

		idle = times[3] + times[4]

		if cpuNum == -1 {
			if total > c.total {
				dTotal = total - c.total
			}

			if idle > c.idle {
				dIdle = idle - c.idle
			}

			c.total = total
			c.idle = idle
			c.percent = int(100 * (dTotal - dIdle) / dTotal)
		} else {
			core := &c.cores[cpuNum]

			if total > core.total {
				dTotal = total - core.total
			}

			if idle > core.idle {
				dIdle = idle - core.idle
			}

			core.total = total
			core.idle = idle
			core.percent = int(100 * (dTotal - dIdle) / dTotal)

			if core.percent < 0 {
				core.percent = 0
			}
		}
	}

	return nil
}

// Update forces the cpu metric to update. The returned error will not
// be sent on the channel returned by [CPU.Updated] unlike updates that
// happen automatically every update interval.
func (c *CPU) Update() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.flags.Has(cpuUsage) {
		if err := c.updateUsage(); err != nil {
			log.WarnError("can't update CPU usage", err)

			c.flags &^= cpuUsage
		}
	}

	if c.temp != nil {
		c.temp.Read()
	}

	for i := range c.temps {
		c.temps[i].Read()
	}

	for i := range c.cores {
		c.cores[i].freq.Read()
	}

	return
}

// Updated returns the channel that updates will be sent on. A received value
// of [ErrNoChange] indicates there were no changes between updates. Any other non-nil
// error is the first error encountered during updating and indicates a failed update.
func (c *CPU) Updated() <-chan error {
	return c.ch
}

// Stop stops the CPU from continuing to update. Once stopped, the CPU
// may not be restarted.
func (c *CPU) Stop() {
	c.mu.Lock()

	if c.stop != nil {
		c.stop()
	}

	c.mu.Unlock()
}

// String implements [fmt.Stringer] and returns a string representing the CPU
// in the form of:
//
//	Name
//	# cores
func (c *CPU) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return fmt.Sprintf("%s\n%d cores", c.Name, len(c.cores))
}

func (c *cpuCore) AppendText(b []byte, flags cpuFlag) []byte {
	b = append(b, "{\"id\": "...)
	b = strconv.AppendInt(b, int64(c.logical), 10)

	if c.temp != nil {
		b = append(b, ", \"temperature\": "...)
		b = byteutil.AppendDecimal(b, c.temp.Value(), 3)
	}

	if flags.Has(cpuFrequency) {
		b = append(b, ", \"frequency\": "...)
		b = byteutil.AppendDecimal(b, c.freq.Curr(), 6)
	}

	if flags.Has(cpuUsage) {
		b = append(b, ", \"usage\": "...)
		b = strconv.AppendInt(b, int64(c.percent), 10)
	}

	return append(b, '}')
}

// AppendText implements [encoding/TextAppender] and appends the JSON-encoded
// representation of c to b.
func (c *CPU) AppendText(b []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	b = append(b, "{\"name\": \""...)
	b = append(b, c.Name...)
	b = append(b, '"')
	temp, freq := c.selectFn()

	if c.temp != nil {
		b = append(b, ", \"temperature\": "...)
		b = byteutil.AppendDecimal(b, temp, 3)
	}

	if c.flags.Has(cpuFrequency) {
		b = append(b, ", \"frequency\": "...)
		b = byteutil.AppendDecimal(b, freq, 6)
	}

	if c.flags.Has(cpuTemperature | cpuFrequency) {
		b = append(b, ", \"selection_mode\": \""...)
		b = append(b, c.selectMode...)
		b = append(b, '"')
	}

	if c.flags.Has(cpuUsage) {
		b = append(b, ", \"usage\": "...)
		b = strconv.AppendInt(b, int64(c.percent), 10)
	}

	b = append(b, ", \"cores\": ["...)

	for i := range c.cores {
		b = c.cores[i].AppendText(b, c.flags)

		if i < len(c.cores)-1 {
			b = append(b, ',', ' ')
		}
	}

	return append(b, ']', '}'), nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [CPU.AppendText](nil).
func (c *CPU) MarshalJSON() ([]byte, error) {
	return c.AppendText(nil)
}

// SelectAuto returns the package temperature and frequency of the first core.
func (c *CPU) SelectAuto() (temp, freq int64) {
	if c.temp == nil {
		return c.SelectFirst()
	}

	temp = c.temp.Value()

	if len(c.cores) > 0 {
		freq = c.cores[0].freq.Curr()
	}

	return
}

// SelectMin returns the temperature and frequency of the first core.
func (c *CPU) SelectFirst() (temp, freq int64) {
	if len(c.cores) == 0 {
		return
	}

	if c.cores[0].temp != nil {
		temp = c.cores[0].temp.Value()
	}

	freq = c.cores[0].freq.Curr()

	return
}

// SelectMin returns the average temperature and frequency of all cores.
func (c *CPU) SelectAvg() (temp, freq int64) {
	for i := range c.cores {
		if c.cores[i].temp != nil {
			temp += c.cores[i].temp.Value()
		}

		freq += c.cores[i].freq.Curr()
	}

	temp /= int64(len(c.cores))
	freq /= int64(len(c.cores))

	return
}

// SelectMin returns the maximum temperature and frequency of all cores.
func (c *CPU) SelectMax() (temp, freq int64) {
	for i := range c.cores {
		if c.cores[i].temp != nil {
			if t := c.cores[i].temp.Value(); t > temp {
				temp = t
			}
		}

		if f := c.cores[i].freq.Curr(); f > freq {
			freq = f
		}
	}

	return
}

// SelectMin returns the minimum temperature and frequency of all cores.
func (c *CPU) SelectMin() (temp, freq int64) {
	for i := range c.cores {
		if c.cores[i].temp != nil {
			if t := c.cores[i].temp.Value(); t < temp || temp == 0 {
				temp = t
			}
		}

		if f := c.cores[i].freq.Curr(); f < freq || freq == 0 {
			freq = f
		}
	}

	return
}

// SelectRand returns the temperature and frequency of a random core.
func (c *CPU) SelectRand() (temp, freq int64) {
	i := c.rand.IntN(len(c.cores))
	if c.cores[i].temp != nil {
		temp = c.cores[i].temp.Value()
	}

	freq = c.cores[i].freq.Curr()

	return
}

func (c *CPU) setSelectionMode(mode string) {
	switch mode {
	case "", "auto":
		c.selectMode = "auto"
		c.selectFn = c.SelectAuto
	case "first":
		c.selectMode = "first"
		c.selectFn = c.SelectFirst
	case "avg", "average":
		c.selectMode = "average"
		c.selectFn = c.SelectAvg
	case "max", "maximum":
		c.selectMode = "maximum"
		c.selectFn = c.SelectMax
	case "min", "minimum":
		c.selectMode = "minimum"
		c.selectFn = c.SelectMin
	case "rand", "random":
		c.selectMode = "random"
		c.selectFn = c.SelectRand
		if c.rand != nil {
			c.rand = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		}
	}
}

func (c *CPU) SetSelectionMode(mode string) {
	c.mu.Lock()
	c.setSelectionMode(strings.ToLower(mode))
	c.mu.Unlock()
}
