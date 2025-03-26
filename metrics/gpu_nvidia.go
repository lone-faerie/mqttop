//go:build !nogpu

package metrics

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"golang.org/x/sync/errgroup"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/sysfs"
	"github.com/lone-faerie/mqttop/log"
)

type gpuFlag uint32

const (
	gpuThroughput gpuFlag = 1 << iota
	gpuUtilization
	gpuClock
	gpuMemClock
	gpuPower
	gpuState
	gpuTemperature
	gpuMemory
	gpuMemoryV2
	gpuProcs
	gpuAll = gpuFlag(1<<32-1) &^ gpuMemory
)

func (f gpuFlag) Has(flag gpuFlag) bool {
	return f&flag != 0
}

type nvmlThroughput struct {
	util nvml.PcieUtilCounter
	val  uint32
}

type nvmlProcess struct {
	Pid       uint32
	Cmd       string
	Mem       uint64
	IsCompute bool
}

// GPU implements the [Metric] interface to provide the Nvidia GPU
// metrics. This includes the throughput, usage, clock, power,
// temperature, and memory of the GPU.
type NvidiaGPU struct {
	Name     string
	maxPower uint32
	maxTemp  uint32
	rx       uint32
	tx       uint32
	oldRx    uint32
	oldTx    uint32
	util     nvml.Utilization
	clock    uint32
	memClock uint32
	power    uint32
	state    nvml.Pstates
	temp     uint32
	memTotal uint64
	memFree  uint64
	memUsed  uint64

	memSize byteutil.ByteSize
	procs   []nvmlProcess

	index   int
	flags   gpuFlag
	changes gpuFlag
	device  nvml.Device

	interval time.Duration
	tick     *time.Ticker
	topic    string

	mu        sync.RWMutex
	once      sync.Once
	stop      context.CancelFunc
	ch        chan error
	pcieGroup errgroup.Group
	nvmlOnce  sync.Once
}

// NewGPU returns a new [GPU] initialized from cfg. If there is any error
// encountered while initializing the GPU, a non-nil error that wraps [ErrNotSupported]
// is returned.
//
// NewGPU calls [nvml.Init], which must be followed by a call to [nvml.Shutdown].
// If a non-nil error is returned, this has already been handled. Otherwise, it
// may be done by either calling [NvidiaGPU.Stop] or cancelling the [context.Context]
// given to [NvidiaGPU.Start].
func NewNvidiaGPU(cfg *config.Config) (*NvidiaGPU, error) {
	g := &NvidiaGPU{flags: gpuAll}
	_, err := sysfs.GPUVendor()
	if err != nil {
		return nil, errNotSupported(g.Type(), err)
	}

	if cfg.GPU.Interval > 0 {
		g.interval = cfg.GPU.Interval
	} else {
		g.interval = cfg.Interval
	}
	if cfg.GPU.Topic != "" {
		g.topic = cfg.GPU.Topic
	} else {
		g.topic = "mqttop/metric/gpu"
	}
	g.index = cfg.GPU.Index

	if err := nvml.Init(); err != nvml.SUCCESS {
		log.Debug("Error initializing nvml", "err", err)
		return nil, errNotSupported(g.Type(), err)
	}
	log.Info("nvml initialized")
	if err := g.init(cfg); err != nvml.SUCCESS {
		g.shutdown()
		return nil, errNotSupported(g.Type(), err)
	}
	size, err := byteutil.ParseSize(cfg.GPU.SizeUnit)
	if err != nil {
		size = byteutil.MiB
	}
	g.memSize = size

	return g, nil
}

func (g *NvidiaGPU) init(cfg *config.Config) error {
	dev, err := nvml.DeviceGetHandleByIndex(g.index)
	if err != nvml.SUCCESS {
		return errNotSupported("DeviceGetHandleByIndex", err)
	}
	name, err := dev.GetName()
	if err != nvml.SUCCESS {
		return errNotSupported("GetName", err)
	}
	g.Name = cfg.GPU.FormatName(name)
	pow, err := dev.GetPowerManagementLimit()
	if err != nvml.SUCCESS {
		pow, err = dev.GetPowerManagementDefaultLimit()
	}
	if err == nvml.SUCCESS {
		g.maxPower = pow
	}
	tmp, err := dev.GetTemperatureThreshold(nvml.TEMPERATURE_THRESHOLD_SHUTDOWN)
	if err == nvml.SUCCESS {
		g.maxTemp = tmp
	}

	g.device = dev
	return nvml.SUCCESS
}

// Type returns the metric type, "gpu".
func (g *NvidiaGPU) Type() string {
	return "gpu"
}

// Topic returns the topic to publish cpu metrics to.
func (g *NvidiaGPU) Topic() string {
	return g.topic
}

// SetInterval sets the update interval for the metric.
func (g *NvidiaGPU) SetInterval(d time.Duration) {
	g.mu.Lock()
	if g.tick != nil && d != g.interval {
		g.tick.Reset(d)
	}
	g.interval = d
	g.mu.Unlock()
}

func (g *NvidiaGPU) loop(ctx context.Context) {
	g.mu.Lock()
	g.tick = time.NewTicker(g.interval)
	g.mu.Unlock()

	defer close(g.ch)
	defer g.shutdown()
	var (
		err error
		ch  chan error
	)
	for {
		select {
		case <-ctx.Done():
			return
		case <-g.tick.C:
			err = g.Update()
			if err == ErrNoChange {
				log.Debug("gpu updated, no change")
				break
			}
			log.Debug("gpu updated")
			ch = g.ch
		case ch <- err:
			ch = nil
		}
	}
}

// Start starts the cpu updating. If ctx is cancelled or
// times out, the metric will stop and may not be restarted.
//
// After calling Start, handling of [nvml.Shutdown] should be done
// by cancelling the given metric, instead of calling [NvidiaGPU.Stop].
func (g *NvidiaGPU) Start(ctx context.Context) error {
	if g.interval == 0 {
		log.Warn("GPU interval is 0, not starting")
		return nil
	}
	g.once.Do(func() {
		ctx, g.stop = context.WithCancel(ctx)
		g.ch = make(chan error)
		go g.loop(ctx)
	})
	return nil
}

func (g *NvidiaGPU) getThroughput(u nvml.PcieUtilCounter, p *uint32) (err error) {
	*p, err = g.device.GetPcieThroughput(u)
	if err == nvml.SUCCESS {
		return err
	}
	return nil
}

// Update forces the gpu metric to update. The returned error will not
// be sent on the channel returned by [GPU.Updated] unlike updates that
// happen automatically every update interval.
func (g *NvidiaGPU) Update() error {
	g.mu.Lock()
	var (
		changes gpuFlag
		rx, tx  uint32
	)
	if g.flags.Has(gpuThroughput) {
		g.pcieGroup.Go(func() error {
			return g.getThroughput(nvml.PCIE_UTIL_RX_BYTES, &rx)
		})
		g.pcieGroup.Go(func() error {
			return g.getThroughput(nvml.PCIE_UTIL_TX_BYTES, &tx)
		})
	}
	if g.flags.Has(gpuUtilization) {
		if u, err := g.device.GetUtilizationRates(); err == nvml.SUCCESS {
			if u != g.util {
				changes |= gpuUtilization
			}
			g.util = u
		} else {
			g.flags &^= gpuUtilization
		}
	}
	if g.flags.Has(gpuClock) {
		if c, err := g.device.GetClockInfo(nvml.CLOCK_GRAPHICS); err == nvml.SUCCESS {
			if c != g.clock {
				changes |= gpuClock
			}
			g.clock = c
		} else {
			g.flags &^= gpuClock
		}
	}
	if g.flags.Has(gpuMemClock) {
		if c, err := g.device.GetClockInfo(nvml.CLOCK_MEM); err == nvml.SUCCESS {
			if c != g.memClock {
				changes |= gpuMemClock
			}
			g.memClock = c
		} else {
			g.flags &^= gpuMemClock
		}
	}
	if g.flags.Has(gpuPower) {
		if p, err := g.device.GetPowerUsage(); err == nvml.SUCCESS {
			if p != g.power {
				changes |= gpuPower
			}
			g.power = p
		} else {
			g.flags &^= gpuPower
		}
	}
	if g.flags.Has(gpuState) {
		if s, err := g.device.GetPowerState(); err == nvml.SUCCESS {
			if s != g.state {
				changes |= gpuState
			}
			g.state = s
		} else {
			g.flags &^= gpuState
		}
	}
	if g.flags.Has(gpuTemperature) {
		if t, err := g.device.GetTemperature(nvml.TEMPERATURE_GPU); err == nvml.SUCCESS {
			if t != g.temp {
				changes |= gpuTemperature
			}
			g.temp = t
		} else {
			g.flags &^= gpuTemperature
		}
	}
	if g.flags.Has(gpuMemoryV2) {
		if m, err := g.device.GetMemoryInfo_v2(); err == nvml.SUCCESS {
			if m.Total != g.memTotal && m.Free != g.memFree && m.Used != g.memUsed {
				changes |= gpuMemoryV2
			}
			g.memTotal = m.Total
			g.memFree = m.Free
			g.memUsed = m.Used
		} else {
			g.flags = g.flags&^gpuMemoryV2 | gpuMemory
		}
	}
	if g.flags.Has(gpuMemory) {
		if m, err := g.device.GetMemoryInfo(); err == nvml.SUCCESS {
			if m.Total != g.memTotal && m.Free != g.memFree && m.Used != g.memUsed {
				changes |= gpuMemory
			}
			g.memTotal = m.Total
			g.memFree = m.Free
			g.memUsed = m.Used
		} else {
			g.flags &^= gpuMemory
		}
	}
	if g.flags.Has(gpuThroughput) {
		if err := g.pcieGroup.Wait(); err == nil {
			if rx != g.rx || tx != g.tx {
				changes |= gpuThroughput
			}
			g.rx = rx
			g.tx = tx
		} else {
			g.flags &^= gpuThroughput
		}
	}
	g.mu.Unlock()
	if changes == 0 {
		return ErrNoChange
	}
	return nil
}

// Updated returns the channel that updates will be sent on. A received value
// of [ErrNoChange] indicates there were no changes between updates. Any other non-nil
// error is the first error encountered during updating and indicates a failed update.
func (g *NvidiaGPU) Updated() <-chan error {
	return g.ch
}

func (g *NvidiaGPU) shutdown() {
	g.nvmlOnce.Do(func() {
		nvml.Shutdown()
		log.Info("nvml shutdown")
	})
}

// Stop stops the CPU from continuing to update. Once stopped, the CPU
// may not be restarted.
//
// This will also call [nvml.Shutdown] if the metric has not yet been
// started. After starting, this should be done by cancelling the
// context given to [NvidiaGPU.Start].
func (g *NvidiaGPU) Stop() {
	g.mu.Lock()
	if g.stop != nil {
		g.stop()
	} else if g.device != nil {
		g.shutdown()
	}
	g.mu.Unlock()
}

// String implements [fmt.Stringer]
func (g *NvidiaGPU) String() string {
	return "  " + g.Name
}

// AppendText implements [encoding/TextAppender]
func (g *NvidiaGPU) AppendText(b []byte) ([]byte, error) {
	g.mu.RLock()
	b = append(b, "{\"name\": \""...)
	b = append(b, g.Name...)
	b = append(b, '"')
	if g.flags.Has(gpuThroughput) {
		b = append(b, ", \"rx\": "...)
		b = strconv.AppendUint(b, uint64(g.rx), 10)
		b = append(b, ", \"tx\": "...)
		b = strconv.AppendUint(b, uint64(g.tx), 10)
	}
	if g.flags.Has(gpuUtilization) {
		b = append(b, ", \"utilization\": {\"gpu\": "...)
		b = strconv.AppendUint(b, uint64(g.util.Gpu), 10)
		b = append(b, ", \"memory\": "...)
		b = strconv.AppendUint(b, uint64(g.util.Memory), 10)
		b = append(b, '}')
	}
	if g.flags.Has(gpuClock) {
		b = append(b, ", \"clock\": "...)
		b = strconv.AppendUint(b, uint64(g.clock), 10)
	}
	if g.flags.Has(gpuMemClock) {
		b = append(b, ", \"memClock\": "...)
		b = strconv.AppendUint(b, uint64(g.memClock), 10)
	}
	if g.flags.Has(gpuPower) {
		b = append(b, ", \"power\": "...)
		b = byteutil.AppendDecimal(b, int64(g.power), 3)
		b = append(b, ", \"maxPower\": "...)
		b = byteutil.AppendDecimal(b, int64(g.maxPower), 3)
	}
	if g.flags.Has(gpuTemperature) {
		b = append(b, ", \"temperature\": "...)
		b = strconv.AppendUint(b, uint64(g.temp), 10)
		b = append(b, ", \"maxTemp\": "...)
		b = strconv.AppendInt(b, int64(g.maxTemp), 10)
	}
	if g.flags.Has(gpuMemoryV2 | gpuMemory) {
		b = append(b, ", \"memory\": {\"total\": "...)
		b = byteutil.AppendSize(b, g.memTotal, g.memSize)
		b = append(b, ", \"free\": "...)
		b = byteutil.AppendSize(b, g.memFree, g.memSize)
		b = append(b, ", \"used\": "...)
		b = byteutil.AppendSize(b, g.memUsed, g.memSize)
		b = append(b, '}')
	}
	b = append(b, '}')
	g.mu.RUnlock()
	return b, nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [GPU.AppendText](nil).
func (g *NvidiaGPU) MarshalJSON() ([]byte, error) {
	return g.AppendText(nil)
}

func appendGPU(m []Metric, cfg *config.Config) []Metric {
	if gpu, err := NewNvidiaGPU(cfg); err == nil {
		m = append(m, gpu)
	}
	return m
}
