package metrics

import (
	"context"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/file"
	"github.com/lone-faerie/mqttop/procfs"
	"github.com/lone-faerie/mqttop/sysfs"
)

// Disk holds the data for each disk monitored by [Disks]
type Disk struct {
	procfs.Mount
	sysfs.BlockIO
	Name   string
	size   byteutil.ByteSize
	total  uint64
	free   uint64
	used   uint64
	reads  int64
	writes int64
	ticks  int64
	showIO bool

	err error
}

// Disks implements the [Metric] interface to provide the system disks
// metrics. This includes the total, free, and used sizes and read and
// write io of each disk.
type Disks struct {
	disks  map[string]*Disk
	showIO bool

	cfg      *config.DisksConfig
	interval time.Duration
	tick     *time.Ticker
	topic    string

	rescanInterval time.Duration
	rescanTick     *time.Ticker

	mu   sync.RWMutex
	once sync.Once
	stop context.CancelFunc
	ch   chan error
}

func (d *Disks) newDisk(mnt *procfs.Mount, cfg *config.DiskConfig) *Disk {
	disk := &Disk{Mount: *mnt}

	if cfg != nil && cfg.Name != "" {
		disk.Name = cfg.Name
	} else if len(disk.Mnt) == 1 && disk.Mnt[0] == filepath.Separator {
		disk.Name = "root"
	} else {
		disk.Name = filepath.Base(disk.Mnt)
	}

	if d.showIO || (cfg != nil && cfg.ShowIO) {
		disk.BlockIO = sysfs.BlockStat(mnt)
		disk.showIO = disk.BlockIO.IsValid()
	}

	return disk
}

// NewCPU returns a new [Disks] initialized from cfg. If there is any error
// encountered while initializing the Disks, a non-nil error that wraps
// [ErrNotSupported] is returned.
func NewDisks(cfg *config.Config) (*Disks, error) {
	d := &Disks{cfg: &cfg.Disks}

	if err := d.rescan(true); err != nil {
		return nil, errNotSupported(d.Type(), err)
	}

	log.Info("Found disks", "count", len(d.disks))

	if cfg.Disks.Interval > 0 {
		d.interval = cfg.Disks.Interval
	} else {
		d.interval = cfg.Interval
	}

	if cfg.Disks.Topic != "" {
		d.topic = cfg.Disks.Topic
	} else if cfg.BaseTopic != "" {
		d.topic = cfg.BaseTopic + "/metric/disks"
	} else {
		d.topic = "mqttop/metric/disks"
	}

	if cfg.Disks.RescanInterval > 0 {
		d.rescanInterval = cfg.Disks.RescanInterval
	}

	d.showIO = cfg.Disks.ShowIO

	return d, nil
}

// Type returns the metric type, "disks".
func (d *Disks) Type() string {
	return "disks"
}

// Topic returns the topic to publish disks metrics to.
func (d *Disks) Topic() string {
	return d.topic
}

// SetInterval sets the update interval for the metric.
func (dsk *Disks) SetInterval(d time.Duration) {
	dsk.mu.Lock()

	if dsk.tick != nil && d != dsk.interval {
		dsk.tick.Reset(d)
	}

	dsk.interval = d

	dsk.mu.Unlock()
}

func (d *Disks) loop(ctx context.Context) {
	d.mu.Lock()

	d.tick = time.NewTicker(d.interval)

	if d.rescanInterval > 0 {
		d.rescanTick = time.NewTicker(d.rescanInterval)
	}

	d.mu.Unlock()

	defer d.tick.Stop()

	var (
		err     error
		ch      chan error
		rescanC <-chan time.Time
	)

	if d.rescanTick != nil {
		rescanC = d.rescanTick.C
		defer d.rescanTick.Stop()
	}

	defer close(d.ch)

	log.Debug("disks started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.tick.C:
			err = d.Update()
			if err == ErrNoChange {
				log.Debug("disks updated, no change")
			} else {
				log.Debug("disks updated", "err", err)
			}

			ch = d.ch
		case <-rescanC:
			err = d.Rescan()
			if err == nil {
				select {
				case <-ctx.Done():
					return
				case d.ch <- ErrRescanned:
				}
			} else if err != ErrNoChange {
				ch = d.ch
				break
			}

			select {
			case <-d.tick.C:
				err = d.Update()
				if err == ErrNoChange {
					log.Debug("disks updated, no change")

					err = nil
				} else {
					log.Debug("disks updated", "err", err)
				}

				ch = d.ch
			default:
			}
		case ch <- err:
			ch = nil
		}
	}
}

// Start starts the disks updating. If ctx is cancelled or
// times out, the metric will stop and may not be restarted.
func (d *Disks) Start(ctx context.Context) (err error) {
	if d.interval == 0 {
		log.Warn("Disks interval is 0, not starting")
		return
	}

	d.once.Do(func() {
		ctx, d.stop = context.WithCancel(ctx)
		d.ch = make(chan error)

		go d.loop(ctx)
	})

	return
}

func (d *Disks) rescan(firstRun bool) error {
	mnts, err := procfs.MountInfo(d.cfg.UseFSTab)
	if err != nil {
		return err
	}

	log.Debug("procfs.MountInfo", "count", len(mnts))

	if firstRun {
		d.disks = make(map[string]*Disk, len(mnts))
	}

	var changed bool

	for name, mnt := range mnts {
		if d.cfg.Excluded(name) {
			continue
		}

		if _, ok := d.disks[name]; !ok {
			dcfg := d.cfg.ConfigFor(name)
			disk := d.newDisk(mnt, dcfg)

			if err := disk.Update(); err != nil {
				log.Error("can't add disk", err, "path", disk.Mnt)
				continue
			}

			if dcfg != nil && dcfg.SizeUnit != "" {
				size, err := byteutil.ParseSize(dcfg.SizeUnit)
				if err != nil {
					size = byteutil.SizeOf(disk.total >> 2)
				}

				disk.size = size
			} else {
				disk.size = byteutil.SizeOf(disk.total >> 2)
			}

			if firstRun {
				disk.used = 0
			}

			d.disks[name] = disk
			changed = true
		}
	}

	if firstRun {
		return nil
	}

	for name := range d.disks {
		if _, ok := mnts[name]; ok {
			continue
		}

		delete(d.disks, name)

		changed = true
	}

	if !changed {
		return ErrNoChange
	}

	return nil
}

// Rescan rescans the system for any new or removed disks.
func (d *Disks) Rescan() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.rescan(false)
}

// Update forces the disks metric to update. The returned error will not
// be sent on the channel returned by [Disks.Updated] unlike updates that
// happen automatically every update interval.
func (d *Disks) Update() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	var group errgroup.Group

	for name := range d.disks {
		group.Go(d.disks[name].Update)
	}

	return group.Wait()
}

// Updated returns the channel that updates will be sent on. A received value
// of [ErrNoChange] indicates there were no changes between updates and a value of
// [ErrRescanned] indicates a change from rescanning. Any other non-nil error is the
// first error encountered during updating and indicates a failed update.
func (d *Disks) Updated() <-chan error {
	return d.ch
}

// Stop stops the Disks from continuing to update. Once stopped, the Disks
// may not be restarted.
func (d *Disks) Stop() {
	d.mu.Lock()

	if d.stop != nil {
		d.stop()
	}

	d.mu.Unlock()
}

// String implements [fmt.Stringer] and returns a string representing the disks
// in the form of:
//
//	name1 (mnt1)
//	  size
//	name2 (mnt2)
//	  size
func (d *Disks) String() string {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var b strings.Builder

	first := true

	for _, disk := range d.disks {
		if !first {
			b.WriteByte('\n')
		}

		b.WriteString(disk.Name)
		b.Write([]byte{' ', '('})
		b.WriteString(disk.Mnt)
		b.Write([]byte{')', '\n', ' ', ' '})
		byteutil.WriteSize(&b, disk.total, disk.size)

		first = false
	}

	return b.String()
}

// AppendText implements [encoding/TextAppender] and appends the JSON-encoded
// representation of d to b.
func (d *Disks) AppendText(b []byte) ([]byte, error) {
	b = append(b, '{')

	first := true

	for _, disk := range d.disks {
		if disk.err != nil {
			continue
		}

		if !first {
			b = append(b, ',', ' ')
		}

		b = append(b, '"')
		b = append(b, disk.Name...)
		b = append(b, "\": {\"mnt\": \""...)
		b = append(b, disk.Mnt...)
		b = append(b, "\", \"total\": "...)
		b = byteutil.AppendSize(b, disk.total, disk.size)
		b = append(b, ", \"free\": "...)
		b = byteutil.AppendSize(b, disk.free, disk.size)
		b = append(b, ", \"used\": "...)
		b = byteutil.AppendSize(b, disk.used, disk.size)

		if disk.showIO {
			b = append(b, ", \"reads\": "...)
			b = strconv.AppendInt(b, disk.reads, 10)
			b = append(b, ", \"writes\": "...)
			b = strconv.AppendInt(b, disk.writes, 10)
		}

		b = append(b, '}')

		first = false
	}

	return append(b, '}'), nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [CPU.AppendText](nil).
func (d *Disks) MarshalJSON() ([]byte, error) {
	return d.AppendText(nil)
}

// Update forces the individual disk to update. The returned error will not
// be sent on the channel returned by [Disks.Updated] unlike updates that
// happen automatically every update interval.
func (d *Disk) Update() (err error) {
	d.err = nil

	stat, err := file.Statfs(d.Mnt)
	if err != nil {
		d.err = err
		return
	}

	total := stat.Blocks * uint64(stat.Frsize)
	free := stat.Bavail * uint64(stat.Frsize)
	used := total - free

	if d.used == used && d.free == free && d.total == total {
		err = ErrNoChange
	}

	d.total = total
	d.free = free
	d.used = used

	if !d.showIO {
		return
	}

	r, w, t, e := d.BlockIO.Read()
	if e != nil {
		log.WarnError("Can't read block io", err, "mnt", d.Mnt)
		d.showIO = false
		d.err = err
		return e
	}

	if err == ErrNoChange && d.reads == r && d.writes == w && d.ticks == t {
		err = ErrNoChange
	}

	d.reads = r
	d.writes = w
	d.ticks = t

	return
}
