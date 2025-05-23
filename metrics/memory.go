package metrics

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/procfs"
)

// Memory implements the [Metric] interface to provide the system memory
// metrics. This includes the total, free, available, and used memory,
// and the total, free, and used swap memory.
type Memory struct {
	total     uint64
	free      uint64
	avail     uint64
	used      uint64
	cached    uint64
	swapTotal uint64
	swapFree  uint64
	swapUsed  uint64

	size        byteutil.ByteSize
	swapSize    byteutil.ByteSize
	includeSwap bool

	interval time.Duration
	tick     *time.Ticker
	topic    string

	mu   sync.RWMutex
	once sync.Once
	stop context.CancelFunc
	ch   chan error
}

// NewMemory returns a new [Memory] initialized from cfg. If there is any error
// encountered while initializing the Memory, a non-nil error that wraps [ErrNotSupported]
// is returned.
func NewMemory(cfg *config.Config) (*Memory, error) {
	m := &Memory{includeSwap: cfg.Memory.IncludeSwap}

	if err := m.parseInfo(); err != nil {
		return nil, errNotSupported(m.Type(), err)
	}

	if cfg.Memory.SizeUnit != "" {
		size, err := byteutil.ParseSize(cfg.Memory.SizeUnit)
		if err == nil {
			m.size = size
		}
	}

	if cfg.Memory.Interval > 0 {
		m.interval = cfg.Memory.Interval
	} else {
		m.interval = cfg.Interval
	}

	if cfg.Memory.Topic != "" {
		m.topic = cfg.Memory.Topic
	} else if cfg.BaseTopic != "" {
		m.topic = cfg.BaseTopic + "/metric/memory"
	} else {
		m.topic = "mqttop/metric/memory"
	}

	return m, nil
}

var (
	totalKey = []byte("MemTotal")
	swapKey  = []byte("SwapTotal")
)

func (m *Memory) parseInfo() error {
	info, err := procfs.MemInfo()
	if err != nil {
		return err
	}

	defer info.Close()

	var includeSwap bool

	for {
		line, err := info.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		key, val := byteutil.Field(line)

		if byteutil.Equal(key, totalKey) {
			m.total = byteutil.Btou(val) << 10
			m.size = byteutil.SizeOf(m.total)

			if m.swapTotal > 0 {
				break
			}
		}

		if byteutil.Equal(key, swapKey) {
			includeSwap = true
			m.swapTotal = uint64(byteutil.Btoi(val)) << 10
			m.swapSize = byteutil.SizeOf(m.swapTotal)

			if m.total > 0 {
				break
			}
		}
	}

	m.includeSwap = m.includeSwap && includeSwap

	return nil
}

// Type returns the metric type, "memory".
func (m *Memory) Type() string {
	return "memory"
}

// Topic returns the topic to publish memory metrics to.
func (m *Memory) Topic() string {
	return m.topic
}

// SetInterval sets the update interval for the metric.
func (m *Memory) SetInterval(d time.Duration) {
	m.mu.Lock()

	if m.tick != nil && d != m.interval {
		m.tick.Reset(d)
	}

	m.interval = d

	m.mu.Unlock()
}

func (m *Memory) loop(ctx context.Context) {
	m.mu.Lock()
	m.tick = time.NewTicker(m.interval)
	m.mu.Unlock()

	defer m.tick.Stop()
	defer close(m.ch)

	var (
		err error
		ch  chan error
	)

	log.Debug("memory started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.tick.C:
			err = m.Update()

			log.Debug("memory updated")

			ch = m.ch
		case ch <- err:
			ch = nil
		}
	}
}

// Start starts the memory updating. If ctx is cancelled or
// times out, the metric will stop and may not be restarted.
func (m *Memory) Start(ctx context.Context) (err error) {
	if m.interval == 0 {
		log.Warn("Memory interval is 0, not starting")
		return
	}

	m.once.Do(func() {
		ctx, m.stop = context.WithCancel(ctx)
		m.ch = make(chan error)

		go m.loop(ctx)
	})

	return
}

// Update forces the memory metric to update. The returned error will not
// be sent on the channel returned by [Memory.Updated] unlike updates that
// happen automatically every update interval.
func (m *Memory) Update() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	info, err := procfs.MemInfo()
	if err != nil {
		return err
	}

	defer info.Close()

	var gotAvailable bool

	for {
		line, err := info.ReadLine()
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		key, val := byteutil.Field(line)

		if len(key) > 0 && key[0] == 'D' {
			break
		}

		switch string(key) {
		case "MemFree":
			m.free = byteutil.Btou(val) << 10
		case "MemAvailable":
			m.avail = byteutil.Btou(val) << 10
			gotAvailable = true
		case "Cached":
			m.cached = byteutil.Btou(val) << 10
		case "SwapTotal":
			if m.includeSwap {
				m.swapTotal = byteutil.Btou(val) << 10
			}
		case "SwapFree":
			if m.includeSwap {
				m.swapFree = byteutil.Btou(val) << 10
			}
		}
	}

	if !gotAvailable {
		m.avail = m.free + m.cached
	}

	if m.avail > m.total {
		m.used = m.total - m.free
	} else {
		m.used = m.total - m.avail
	}

	if m.swapTotal > 0 {
		m.swapUsed = m.swapTotal - m.swapFree
	}

	return nil
}

// Updated returns the channel that updates will be sent on. A received value
// of [ErrNoChange] indicates there were no changes between updates. Any other non-nil
// error is the first error encountered during updating and indicates a failed update.
func (m *Memory) Updated() <-chan error {
	return m.ch
}

// Stop stops the CPU from continuing to update. Once stopped, the CPU
// may not be restarted.
func (m *Memory) Stop() {
	m.mu.Lock()

	if m.stop != nil {
		m.stop()
	}

	m.mu.Unlock()
}

// String implements [fmt.Stringer] and returns a string representing the
// total amount of memory.
func (m *Memory) String() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var b strings.Builder

	byteutil.WriteSize(&b, m.total, m.size)

	return b.String()
}

// AppendText implements [encoding/TextAppender] and appends the JSON-encoded
// representation of m to b.
func (m *Memory) AppendText(b []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b = append(b, "{\"total\": "...)
	b = byteutil.AppendSize(b, m.total, m.size)
	b = append(b, ", \"used\": "...)
	b = byteutil.AppendSize(b, m.used, m.size)
	b = append(b, ", \"available\": "...)
	b = byteutil.AppendSize(b, m.avail, m.size)
	b = append(b, ", \"cached\": "...)
	b = byteutil.AppendSize(b, m.cached, m.size)
	b = append(b, ", \"free\": "...)
	b = byteutil.AppendSize(b, m.free, m.size)

	if m.swapTotal > 0 {
		b = append(b, ", \"swapTotal\": "...)
		b = byteutil.AppendSize(b, m.swapTotal, m.swapSize)
		b = append(b, ", \"swapUsed\": "...)
		b = byteutil.AppendSize(b, m.swapUsed, m.swapSize)
		b = append(b, ", \"swapFree\": "...)
		b = byteutil.AppendSize(b, m.swapFree, m.swapSize)
	}

	return append(b, '}'), nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [CPU.AppendText](nil).
func (m *Memory) MarshalJSON() ([]byte, error) {
	return m.AppendText(nil)
}
