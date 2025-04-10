package metrics

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/sysfs"
)

// NetInterface holds the data for each network interface monitored
// by [Net].
type NetInterface struct {
	net.Interface
	IP           netip.Addr
	Upload       uint64
	Download     uint64
	UploadRate   uint64
	DownloadRate uint64
	rate         byteutil.ByteRate
	lastUpdate   time.Time
	lastTx       uint64
	lastRx       uint64
}

// Running returns true if the interface is running, else false.
func (iface *NetInterface) Running() bool {
	return iface.Interface.Flags&net.FlagRunning != 0
}

// Net implements the [Metric] interface to provide the system network
// metrics. This includes the ip address, rx and tx throughput of
// each network interface.
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

// NewNet returns a new [Net] initialized from cfg. If there is any error
// encountered while initializing the Net, a non-nil error that wraps [ErrNotSupported]
// is returned.
func NewNet(cfg *config.Config) (*Net, error) {
	n := &Net{cfg: &cfg.Net}

	if err := n.parseInterfaces(true); err != nil {
		return nil, errNotSupported(n.Type(), err)
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

func ipAddr(addr string) (netip.Addr, error) {
	if a, err := netip.ParseAddr(addr); err == nil {
		return a, nil
	}

	if ap, err := netip.ParseAddrPort(addr); err == nil {
		return ap.Addr(), nil
	}

	p, err := netip.ParsePrefix(addr)
	if err != nil {
		return netip.Addr{}, err
	}

	return p.Addr(), nil
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
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	if firstRun {
		n.interfaces = make(map[string]*NetInterface, len(interfaces))
	}

	var changed bool

	for i := range interfaces {
		addrs, err := interfaces[i].Addrs()
		if err != nil {
			return err
		}

		ifname := interfaces[i].Name
		if iface, ok := n.interfaces[ifname]; !ok || !firstRun {
			var ip netip.Addr

			if len(addrs) > 0 {
				ip, err = ipAddr(addrs[0].String())
				if err != nil {
					return err
				}
			}

			var (
				ratestr string
			)

			for j := range n.cfg.Include {
				if n.cfg.Include[j].Interface != ifname {
					continue
				}

				ifname = n.cfg.Include[j].FormatName(ifname)
				ratestr = n.cfg.Include[j].RateUnit

				break
			}

			if n.skipInterface(interfaces[i].Name) {
				if !firstRun {
					delete(n.interfaces, ifname)
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

				log.Debug("Adding interface", "name", ifname)

				n.interfaces[ifname] = &NetInterface{
					Interface: interfaces[i],
					IP:        ip,
					rate:      rate,
				}
				changed = true
			} else {
				iface.Interface = interfaces[i]

				if ip != iface.IP {
					iface.IP = ip
				}
			}
		}
	}

	if firstRun {
		return nil
	}

	for iface := range n.interfaces {
		if !slices.ContainsFunc(interfaces, func(i net.Interface) bool {
			return i.Name == iface
		}) {
			log.Debug("Deleting interface", "name", iface)
			delete(n.interfaces, iface)

			changed = true
		}
	}

	if !changed {
		return ErrNoChange
	}

	return nil
}

// Type returns the metric type, "net".
func (n *Net) Type() string {
	return "net"
}

// Topic returns the topic to publish net metrics to.
func (n *Net) Topic() string {
	return n.topic
}

// SetInterval sets the update interval for the metric.
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

	var group errgroup.Group

	for name, iface := range n.interfaces {
		log.Debug("Updating interface", "name", name)
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

// String implements [fmt.Stringer] and returns a string representing the net
// in the form of:
//
//	# interfaces (# running)
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

// AppendText implements [encoding/TextAppender] and appends the JSON-encoded
// representation of n to b.
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

		if iface.IP.IsValid() {
			b = append(b, "\"ip\": \""...)
			b = iface.IP.AppendTo(b)
			b = append(b, '"', ',', ' ')
		}

		if !iface.Running() {
			b = append(b[:len(b)-2], '}')
			first = false

			continue
		}

		b = append(b, "\"download\": "...)
		b = strconv.AppendUint(b, iface.Download, 10)
		b = append(b, ", \"upload\": "...)
		b = strconv.AppendUint(b, iface.Upload, 10)

		size := byteutil.ByteSize(iface.rate)

		b = append(b, ", \"download_rate\": "...)
		b = byteutil.AppendSize(b, iface.DownloadRate, size)
		b = append(b, ", \"upload_rate\": "...)
		b = byteutil.AppendSize(b, iface.UploadRate, size)
		b = append(b, '}')

		first = false
	}

	return append(b, '}'), nil
}

// MarshalJSON implements [json.Marshaler] and is equivalent to [CPU.AppendText](nil).
func (n *Net) MarshalJSON() ([]byte, error) {
	return n.AppendText(nil)
}

// Update forces the individual network interface to update. The returned
// error will not be sent on the channel returned by [Net.Updated] unlike
// updates that happen automatically every update interval.
func (iface *NetInterface) Update() error {
	rx, tx, err := sysfs.NetStatistics(iface.Name)
	if err != nil {
		return &os.PathError{Op: "open", Path: iface.Name, Err: err}
	}

	now := time.Now()
	iface.Download = rx - iface.lastRx
	iface.Upload = tx - iface.lastTx
	iface.lastRx = rx
	iface.lastTx = tx
	delta := uint64(now.Sub(iface.lastUpdate) / time.Second)

	if delta > 0 {
		iface.DownloadRate = 100 * iface.Download / delta
		iface.UploadRate = 100 * iface.Upload / delta
	}

	iface.lastUpdate = now

	return nil
}
