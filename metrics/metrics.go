package metrics

import (
	"context"
	"encoding"
	"encoding/json"
	"slices"
	"time"

	"github.com/lone-faerie/mqttop/config"
)

// Metric is the interface for providing metrics over mqtt.
type Metric interface {
	// Type returns a constant string representing the type of the metric.
	Type() string
	// Topic returns the topic the metric will be published to.
	Topic() string
	// SetInterval sets the update interval of the metric.
	SetInterval(time.Duration)
	// Start starts listening for updates of the metric. This may only be called once
	// per metric. Any calls to Start after stopping the metric will do nothing.
	Start(context.Context) error
	// Update forces the metric to update regardless of the update interval.
	Update() error
	// Updated returns the channel updates will be published to every update interval.
	// There may not be anything sent on the channel if there were no changes between
	// updates, and a nil value indicates a successful update.
	Updated() <-chan error
	// Stop stops the metric from listening to updates. The metric may not be restarted
	// after stopping.
	Stop()

	String() string
	encoding.TextAppender
	json.Marshaler
}

// NewMetrics returns a slice of all the metrics enabled in the given config.
// If any metric returns an error, it is simply ignored and will not be in the slice.
func New(cfg *config.Config) []Metric {
	var m []Metric
	if cfg.CPU.Enabled {
		if cpu, err := NewCPU(cfg); err == nil {
			m = append(m, cpu)
		}
	}
	if cfg.Memory.Enabled {
		if mem, err := NewMemory(cfg); err == nil {
			m = append(m, mem)
		}
	}
	if cfg.Disks.Enabled {
		if disks, err := NewDisks(cfg); err == nil {
			m = append(m, disks)
		}
	}
	if cfg.Net.Enabled {
		if net, err := NewNet(cfg); err == nil {
			m = append(m, net)
		}
	}
	if cfg.Battery.Enabled {
		if bat, err := NewBattery(cfg); err == nil {
			m = append(m, bat)
		}
	}
	if len(cfg.Dirs) > 0 {
		m = slices.Grow(m, len(cfg.Dirs))
	}
	for i := range cfg.Dirs {
		if dir, err := newDir(&cfg.Dirs[i], cfg); err == nil {
			m = append(m, dir)
		}
	}
	if cfg.GPU.Enabled {
		m = appendGPU(m, cfg)
	}
	return m
}
