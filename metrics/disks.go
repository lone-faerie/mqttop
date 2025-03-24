package metrics

import (
	"context"
	"golang.org/x/sys/unix"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/procfs"
	"github.com/lone-faerie/mqttop/internal/sysfs"
)

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
}

type Disks struct {
	disks  map[string]*Disk
	showIO bool

	cfg      *config.Config
	interval time.Duration
	tick     *time.Ticker
	topic    string

	rescanInterval time.Duration
	rescanTick     *time.Ticker

	mu    sync.RWMutex
	once  sync.Once
	group errgroup.Group
	stop  context.CancelFunc
	ch    chan error
}

func newDisk(mnt *procfs.Mount, cfg *config.Config, dcfg *config.DiskConfig) *Disk {
	d := &Disk{Mount: *mnt}

	if dcfg != nil && dcfg.Name != "" {
		d.Name = dcfg.Name
	} else if len(d.Mnt) == 1 && d.Mnt[0] == filepath.Separator {
		d.Name = "root"
	} else {
		d.Name = filepath.Base(d.Mnt)
	}
	if cfg.Disks.ShowIO || (dcfg != nil && dcfg.ShowIO) {
		d.BlockIO = sysfs.BlockStat(mnt)
		d.showIO = d.BlockIO.IsValid()
	}
	return d
}

func NewDisks(cfg *config.Config) (*Disks, error) {
	d := &Disks{cfg: cfg}
	if err := d.rescan(true); err != nil {
		return nil, errNotSupported(d.Type(), err)
	}
	if cfg.Disks.Interval > 0 {
		d.interval = cfg.Disks.Interval
	} else {
		d.interval = cfg.Interval
	}
	if cfg.Disks.Topic != "" {
		d.topic = cfg.Disks.Topic
	} else {
		d.topic = "mqttop/metric/disks"
	}
	if cfg.Disks.RescanInterval > 0 {
		d.rescanInterval = cfg.Disks.RescanInterval
	}
	d.showIO = cfg.Disks.ShowIO

	return d, nil
}

func (d *Disks) Type() string {
	return "disks"
}

func (d *Disks) Topic() string {
	return d.topic
}

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
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.tick.C:
			err = d.Update()
			if err == ErrNoChange {
				log.Debug("disks updated, no change")
				break
			}
			log.Debug("Disks updated")
			ch = d.ch
		case <-rescanC:
			d.Rescan()
			select {
			case <-d.tick.C:
				err = d.Update()
				if err == ErrNoChange {
					log.Debug("disks updated, no change")
					break
				}
				log.Debug("disks updated")
				ch = d.ch
			default:
			}
		case ch <- err:
			ch = nil
		}
	}
}

func (d *Disks) Start(ctx context.Context) (err error) {
	if d.interval == 0 {
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
	mnts, err := procfs.MountInfo(d.cfg.Disks.UseFSTab)
	if err != nil {
		return err
	}
	if firstRun {
		d.disks = make(map[string]*Disk, len(mnts))
	}
	for name, mnt := range mnts {
		if d.cfg.Disks.Excluded(name) {
			continue
		}
		if _, ok := d.disks[name]; !ok {
			dcfg := d.cfg.Disks.ConfigFor(name)
			disk := newDisk(mnt, d.cfg, dcfg)
			if err := disk.Update(); err != nil {
				continue
			}
			if dcfg != nil && dcfg.SizeUnit != "" {
				size, err := byteutil.ParseSize(dcfg.SizeUnit)
				if err != nil {
					size = byteutil.SizeOf(disk.total)
				}
				disk.size = size
			} else {
				disk.size = byteutil.SizeOf(disk.total)
			}
			d.disks[name] = disk
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
	}
	return nil
}

func (d *Disks) Rescan() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.rescan(false)
}

func (d *Disks) Update() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for name := range d.disks {
		d.group.Go(d.disks[name].Update)
	}
	return d.group.Wait()
}

func (d *Disks) Updated() <-chan error {
	return d.ch
}

func (d *Disks) Stop() {
	d.mu.Lock()
	if d.stop != nil {
		d.stop()
	}
	d.mu.Unlock()
}

func (d *Disks) String() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b strings.Builder
	first := true
	for _, disk := range d.disks {
		if !first {
			b.WriteByte('\n')
		}
		b.Write([]byte{' ', ' '})
		b.WriteString(disk.Name)
		b.Write([]byte{' ', '('})
		b.WriteString(disk.Mnt)
		b.Write([]byte{')', '\n', ' ', ' ', ' ', ' '})
		byteutil.WriteSize(&b, disk.total, disk.size)
		first = false
	}
	return b.String()
}

func (d *Disks) AppendText(b []byte) ([]byte, error) {
	b = append(b, '{')
	first := true
	for _, disk := range d.disks {
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

func (d *Disks) MarshalJSON() ([]byte, error) {
	return d.AppendText(nil)
}

func (d *Disk) update(wg *sync.WaitGroup) error {
	defer wg.Done()
	return d.Update()
}

func (d *Disk) Update() (err error) {
	var stat unix.Statfs_t
	if err = unix.Statfs(d.Mnt, &stat); err != nil {
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
	d.size = byteutil.SizeOf(d.total >> 2)

	if !d.showIO {
		return
	}

	r, w, t, e := d.BlockIO.Read()
	if e != nil {
		return e
	}
	if d.reads == r && d.writes == w && d.ticks == t {
		err = ErrNoChange
	}
	d.reads = r
	d.writes = w
	d.ticks = t
	return
}
