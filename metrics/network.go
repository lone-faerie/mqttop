package metrics

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/sysfs"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

type NetInterface struct {
	name   string
	ip     netip.Addr
	flags  uint16
	rx     uint64
	tx     uint64
	rxRate uint64
	txRate uint64
	rxLast uint64
	txLast uint64
	rate   byteutil.ByteRate

	lastUpdate time.Time
	sockfd     int
}

func (iface *NetInterface) Running() bool {
	return iface.flags&unix.IFF_RUNNING != 0
}

type Net struct {
	interfaces map[string]*NetInterface

	cfg      *config.NetConfig
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

func NewNet(cfg *config.Config) (*Net, error) {
	n := &Net{cfg: &cfg.Net}

	if err := n.parseInterfaces(true); err != nil {
		return nil, err
	}

	if cfg.Net.Interval > 0 {
		n.interval = cfg.Net.Interval
	} else {
		n.interval = cfg.Interval
	}

	if cfg.Net.Topic != "" {
		n.topic = cfg.Net.Topic
	} else if cfg.BaseTopic != "" {
		n.topic = cfg.BaseTopic + "/metric/net"
	} else {
		n.topic = "mqttop/metric/net"
	}

	if cfg.Net.RescanInterval > 0 {
		n.rescanInterval = cfg.Net.RescanInterval
	}

	return n, nil
}

func getAddr4(sock int, ifname string) (addr netip.Addr, err error) {
	i, err := unix.NewIfreq(ifname)
	if err != nil {
		return
	}
	return getAddr4Ifreq(sock, i)
}

func getAddr4Ifreq(sock int, ifreq *unix.Ifreq) (addr netip.Addr, err error) {
	if err = unix.IoctlIfreq(sock, unix.SIOCGIFADDR, ifreq); err != nil {
		return
	}
	in4, err := ifreq.Inet4Addr()
	if err != nil {
		return
	}
	addr = netip.AddrFrom4([4]byte(in4))
	return
}

func getFlagsIfreq(sock int, ifreq *unix.Ifreq) (flags uint16, err error) {
	ifreq.SetUint16(0)
	if err = unix.IoctlIfreq(sock, unix.SIOCGIFFLAGS, ifreq); err != nil {
		return
	}
	flags = ifreq.Uint16()
	return
}

func (n *Net) skipInterface(iface string) bool {
	if slices.Contains(n.cfg.Exclude, iface) {
		return true
	}

	if !n.cfg.OnlyPhysical && n.cfg.IncludeBridge {
		return false
	}

	nd, err := sysfs.NetDevice(iface)
	if err != nil {
		log.Debug("skipInterface", "Error opening", iface)
		return true
	}

	defer nd.Close()

	if slices.ContainsFunc(n.cfg.Include, func(i config.NetIfaceConfig) bool {
		return i.Interface == iface
	}) {
		return false
	} else if len(n.cfg.Include) > 0 {
		return true
	}

	var skip bool

	if n.cfg.OnlyPhysical {
		b := nd.Contains("device")
		skip = skip || !b
	}

	if !n.cfg.IncludeBridge {
		b := nd.Contains("bridge")
		skip = skip || b
	}

	return skip
}

func (n *Net) parseInterfaces(firstRun bool) error {
	dir, err := sysfs.Net()
	if err != nil {
		log.Debug("Error opening /sys/class/net", "err", err)
		return err
	}
	defer dir.Close()

	interfaces, err := dir.ReadNames()
	if err != nil {
		return err
	}
	log.Debug("Read interfaces", "names", interfaces)

	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sock)

	if firstRun {
		n.interfaces = make(map[string]*NetInterface, len(interfaces))
	}

	var changed bool

	for _, name := range interfaces {
		if iface, ok := n.interfaces[name]; !ok || !firstRun {
			addr, err := getAddr4(sock, name)
			if err != nil {
				continue
			}

			var ratestr string

			for i := range n.cfg.Include {
				if n.cfg.Include[i].Interface != name {
					continue
				}

				name = n.cfg.Include[i].FormatName(name)
				ratestr = n.cfg.Include[i].RateUnit
			}

			if n.skipInterface(name) {
				if !firstRun {
					delete(n.interfaces, name)
				}

				continue
			}

			if firstRun || !ok {
				if ratestr == "" {
					ratestr = n.cfg.RateUnit
				}

				rate, err := byteutil.ParseRate(ratestr)
				if err != nil {
					rate = byteutil.MiBps
				}

				log.Debug("Adding interface", "name", name)

				n.interfaces[name] = &NetInterface{
					name: name,
					ip:   addr,
					rate: rate,
				}
				changed = true
			} else {
				if addr != iface.ip {
					iface.ip = addr
				}
			}
		}
	}

	if firstRun {
		return nil
	}

	for name := range n.interfaces {
		if !slices.Contains(interfaces, name) {
			log.Debug("Deleting interface", "name", name)
			delete(n.interfaces, name)

			changed = true
		}
	}

	if !changed {
		return ErrNoChange
	}

	return nil
}

func (n *Net) Type() string {
	return "net"
}

func (n *Net) Topic() string {
	return n.topic
}

func (n *Net) SetInterval(d time.Duration) {
	n.mu.Lock()

	if n.tick != nil && d != n.interval {
		n.tick.Reset(d)
	}

	n.interval = d

	n.mu.Unlock()
}

func (n *Net) loop(ctx context.Context) {
	n.mu.Lock()

	n.tick = time.NewTicker(n.interval)

	if n.rescanInterval > 0 {
		n.rescanTick = time.NewTicker(n.rescanInterval)
	}

	n.mu.Unlock()
	defer n.tick.Stop()

	var (
		err     error
		ch      chan error
		rescanC <-chan time.Time
	)

	if n.rescanTick != nil {
		rescanC = n.rescanTick.C
		defer n.rescanTick.Stop()
	}

	defer close(n.ch)

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.tick.C:
			err = n.Update()

			log.Debug("network updated")

			ch = n.ch
		case <-rescanC:
			err = n.Rescan()
			if err == nil {
				log.Debug("network rescanned")
				select {
				case <-ctx.Done():
					return
				case n.ch <- ErrRescanned:
				}
			} else if err != ErrNoChange {
				ch = n.ch
				break
			} else {
				log.Debug("network rescanned, no change")
			}

			select {
			case <-n.tick.C:
				err = n.Update()

				log.Debug("network updated")

				ch = n.ch
			default:
			}
		case ch <- err:
			ch = nil
		}
	}
}

// Start starts the net updating. If ctx is cancelled or
// times out, the metric will stop and may not be restarted.
func (n *Net) Start(ctx context.Context) (err error) {
	if n.interval == 0 {
		log.Warn("Network interval is 0, not starting")
		return
	}

	n.once.Do(func() {
		ctx, n.stop = context.WithCancel(ctx)
		n.ch = make(chan error)

		go n.loop(ctx)
	})

	return
}

// Rescan rescans the system for any new or removed network interfaces.
func (n *Net) Rescan() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	return n.parseInterfaces(false)
}

// Update forces the net metric to update. The returned error will not
// be sent on the channel returned by [Net.Updated] unlike updates that
// happen automatically every update interval.
func (n *Net) Update() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	sock, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(sock)

	var group errgroup.Group

	for _, iface := range n.interfaces {
		iface.sockfd = sock
		group.Go(iface.Update)
	}

	return group.Wait()
}

// Updated returns the channel that updates will be sent on. A received value
// of [ErrNoChange] indicates there were no changes between updates and a value of
// [ErrRescanned] indicates a change from rescanning. Any other non-nil error is the
// first error encountered during updating and indicates a failed update.
func (n *Net) Updated() <-chan error {
	return n.ch
}

// Stop stops the Net from continuing to update. Once stopped, the Net
// may not be restarted.
func (n *Net) Stop() {
	n.mu.Lock()

	if n.stop != nil {
		n.stop()
	}

	n.mu.Unlock()
}

func (n *Net) String() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var running int

	for _, iface := range n.interfaces {
		if iface.Running() {
			running++
		}
	}

	return fmt.Sprintf("%d interfaces (%d running)", len(n.interfaces), running)
}

func (n *Net) AppendText(b []byte) ([]byte, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	b = append(b, '{')

	first := true

	for name, iface := range n.interfaces {
		if n.cfg.OnlyRunning && !iface.Running() {
			continue
		}

		if !first {
			b = append(b, ',', ' ')
		}

		b = append(b, '"')
		b = append(b, name...)
		b = append(b, "\": {\"running\": "...)

		if iface.Running() {
			b = append(b, "true, "...)
		} else {
			b = append(b, "false, "...)
		}

		if iface.ip.IsValid() {
			b = append(b, "\"ip\": \""...)
			b = iface.ip.AppendTo(b)
			b = append(b, '"', ',', ' ')
		}

		if !iface.Running() {
			b = append(b[:len(b)-2], '}')
			first = false

			continue
		}

		b = append(b, "\"download\": "...)
		b = strconv.AppendUint(b, iface.rx, 10)
		b = append(b, ", \"upload\": "...)
		b = strconv.AppendUint(b, iface.tx, 10)

		size := byteutil.ByteSize(iface.rate)

		b = append(b, ", \"download_rate\": "...)
		b = byteutil.AppendSize(b, iface.rxRate, size)
		b = append(b, ", \"upload_rate\": "...)
		b = byteutil.AppendSize(b, iface.txRate, size)
		b = append(b, '}')

		first = false
	}

	return append(b, '}'), nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [Net.AppendText](nil).
func (n *Net) MarshalJSON() ([]byte, error) {
	return n.AppendText(nil)
}

// Update forces the individual network interface to update. The returned
// error will not be sent on the channel returned by [Net.Updated] unlike
// updates that happen automatically every update interval.
func (iface *NetInterface) Update() error {
	if iface.sockfd != 0 {
		defer func() { iface.sockfd = 0 }()

		ifreq, err := unix.NewIfreq(iface.name)
		if err != nil {
			return err
		}

		ip, err := getAddr4Ifreq(iface.sockfd, ifreq)
		if err != nil {
			return err
		}
		iface.ip = ip

		flags, err := getFlagsIfreq(iface.sockfd, ifreq)
		if err != nil {
			return err
		}
		iface.flags = flags
	}

	rx, tx, err := sysfs.NetStatistics(iface.name)
	if err != nil {
		return &os.PathError{Op: "open", Path: iface.name, Err: err}
	}

	now := time.Now()
	iface.rx = rx - iface.rxLast
	iface.tx = tx - iface.txLast
	iface.rxLast = rx
	iface.txLast = tx
	delta := uint64(now.Sub(iface.lastUpdate) / time.Second)

	if delta > 0 {
		iface.rxRate = 100 * iface.rx / delta
		iface.txRate = 100 * iface.tx / delta
	}

	iface.lastUpdate = now

	return nil
}
