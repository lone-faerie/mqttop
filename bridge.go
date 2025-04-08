package mqttop

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

var logOnce sync.Once

var errNoMetrics = errors.New("no metrics")

// Bridge is the mqtt client that bridges metrics to the mqtt broker.
type Bridge struct {
	client mqtt.Client

	topicPrefix  string
	discoveryCfg *config.DiscoveryConfig
	m            []metrics.Metric
	states       sync.Map

	updates    chan metrics.Metric
	rediscover chan metrics.Metric
	once       sync.Once

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	ready  chan error
	done   chan struct{}
}

// New returns a new Bridge with the provided config and a [mqtt.Client] derived from the config.
// The bridge must have [Bridge.Connect] and [Bridge.Ready] called on it before it may be used.
// This follows the convention of [mqtt.NewClient] as well as waiting for metrics to be ready.
func New(cfg *config.Config) *Bridge {
	opts := cfg.MQTT.ClientOptions()
	client := mqtt.NewClient(opts)
	return NewWithClient(cfg, client)
}

// NewWithClient returns a new Bridge with the provided config and [mqtt.Client].
// The bridge must have [Bridge.Connect] and [Bridge.Ready] called on it before it may be used.
// This follows the convention of [mqtt.NewClient] as well as waiting for metrics to be ready.
func NewWithClient(cfg *config.Config, c mqtt.Client) *Bridge {
	if cfg.MQTT.LogLevel <= log.LevelError {
		mqtt.ERROR = log.ErrorLogger()
	}
	if cfg.MQTT.LogLevel <= log.LevelWarn {
		mqtt.WARN = log.WarnLogger()
	}
	if cfg.MQTT.LogLevel <= log.LevelDebug {
		mqtt.DEBUG = log.DebugLogger()
	}
	if cfg.Discovery.Enabled && cfg.Discovery.DeviceName == "username" {
		cfg.Discovery.DeviceName = cfg.MQTT.Username
	}
	if cfg.Discovery.Availability == "" {
		cfg.Discovery.Availability = cfg.MQTT.BirthWillTopic
	}
	return &Bridge{
		client:       c,
		m:            metrics.New(cfg),
		topicPrefix:  cfg.TopicPrefix,
		discoveryCfg: &cfg.Discovery,
	}
}

func (b *Bridge) AddMetric(m ...metrics.Metric) {
	b.m = append(b.m, m...)
}

func waitToken(ctx context.Context, t mqtt.Token) error {
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	return t.Error()
}

func (b *Bridge) handleMetric(i int, m metrics.Metric) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		switch {
		case strings.HasSuffix(msg.Topic(), "update"):
			go func() {
				payload := string(msg.Payload())
				if d, err := time.ParseDuration(payload); err == nil {
					m.SetInterval(d)
				}
				if err := m.Update(); err == nil {
					b.updates <- m
				}
			}()
		case strings.HasSuffix(msg.Topic(), "stop"):
			go m.Stop()
		}
	}
}

func (b *Bridge) updateStatus(ctx context.Context, m metrics.Metric, state bool) (updated bool) {
	key := m.Topic()
	if updated = b.states.CompareAndSwap(key, !state, state); !updated {
		return
	}
	log.Debug("Status changed", "key", key, "from", !state, "to", state)
	if err := b.publishStatus(ctx, false); err != nil {
		log.WarnError("Unable to update status", err, "metric", m.Type())
		updated = false
	}
	return
}

var statusBuf []byte

func (b *Bridge) publishStatus(ctx context.Context, lwt bool) (err error) {
	var (
		data []byte
		opts = b.client.OptionsReader()
	)
	if ctx == nil {
		ctx = context.Background()
	}
	if lwt {
		data = opts.WillPayload()
	} else {
		data = []byte{'{'}
		first := true
		b.states.Range(func(k, v any) bool {
			if !first {
				data = append(data, ',')
			}
			data = strconv.AppendQuote(data, k.(string))
			data = append(data, ':')
			data = strconv.AppendBool(data, v.(bool))
			first = false
			return true
		})
		data = append(data, '}')
	}
	t := b.client.Publish(opts.WillTopic(), opts.WillQos(), opts.WillRetained(), data)
	return waitToken(ctx, t)
}

func (b *Bridge) publishUpdates(ctx context.Context) {
	var (
		t    mqtt.Token
		done <-chan struct{}
	)
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-b.updates:
			if !ok {
				return
			}
			data, _ := m.AppendText(nil)
			t = b.client.Publish(m.Topic(), 0, false, data)
			done = t.Done()
		case <-done:
			if err := t.Error(); err != nil {
				log.Error("Unable to publish update", err)
			}
			done = nil
		}
	}

}

func metricTopics(m metrics.Metric) map[string]byte {
	return map[string]byte{
		m.Topic() + "/update": 0,
		m.Topic() + "/stop":   0,
	}
}

// Start sets up each metric and begins listening for updates. Any updates that
// return nil errors will be published to the relevant metric's topic.
func (b *Bridge) Start(ctx context.Context) {
	b.once.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		b.ready = make(chan error)
		b.done = make(chan struct{})
		b.updates = make(chan metrics.Metric)
		if b.discoveryCfg.Enabled {
			b.rediscover = make(chan metrics.Metric)
		}
		//		b.states.MakeSize(len(b.m))
		ctx, b.cancel = context.WithCancel(ctx)
		go b.start(ctx)
	})
}

func (b *Bridge) startMetric(ctx context.Context, i int, m metrics.Metric) {
	if m.Topic() == "" {
		log.Debug("No topic, skipping", "metric", m.Type())
		return
	}
	if err := m.Start(ctx); err != nil {
		log.Error("Error starting "+m.Type(), err)
		b.states.Store(m.Topic(), false)
		return
	}
	b.states.Store(m.Topic(), true)
	t := b.client.SubscribeMultiple(metricTopics(m), b.handleMetric(i, m))
	if err := waitToken(ctx, t); err != nil {
		log.Error("Error subscribing to "+m.Topic(), err)
		return
	}
	b.wg.Add(1)
	go func(idx int, metric metrics.Metric) {
		defer b.states.Delete(metric.Topic())
		defer func() {
			metric.Stop()
			b.mu.Lock()
			b.m[idx] = nil
			b.mu.Unlock()
		}()
		defer b.wg.Done()
		ch := metric.Updated()
		if d, ok := metric.(*metrics.Dir); ok {
			log.Info(metric.Type()+" started", "path", d)
		} else {
			log.Info(metric.Type() + " started")
		}
		for err := range ch {
			updated := b.updateStatus(ctx, metric, err == nil || err == metrics.ErrNoChange || err == metrics.ErrRescanned)
			switch err {
			case nil:
				select {
				case <-ctx.Done():
					return
				case b.updates <- metric:
				}
			case metrics.ErrNoChange:
				if !updated {
					break
				}
				select {
				case <-ctx.Done():
					return
				case b.updates <- metric:
				}
			case metrics.ErrRescanned:
				if b.rediscover == nil {
					break
				}
				select {
				case <-ctx.Done():
					return
				case b.rediscover <- metric:
				}
			default:
				log.WarnError(metric.Type()+" not updated", err)
			}
		}
		log.Info(metric.Type() + " done")
	}(i, m)
}

// start starts listening to the metrics.
func (b *Bridge) start(ctx context.Context) {
	log.Debug("Starting")
	defer close(b.ready)
	for i, m := range b.m {
		b.startMetric(ctx, i, m)
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
	if err := b.publishStatus(ctx, false); err != nil {
		log.Error("Unable to publish birth message", err)
	}
	go b.publishUpdates(ctx)
	if b.topicPrefix == "" {
		b.topicPrefix = "mqttop"
	}
	t := b.client.Subscribe(b.topicPrefix+"/bridge/stop", 0, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		b.Disconnect()
	})
	if err := waitToken(ctx, t); err != nil {
		log.Error("Unable to subscribe to stop topic", err)
		b.ready <- err
	}
}

// Ready returns a channel that can be used to wait until all metrics have been started.
// If an error is encountered while starting metrics, it will be sent on this channel.
func (b *Bridge) Ready() <-chan error {
	return b.ready
}

// Done returns a channel that can be used to wait until the bridge has disconnected.
func (b *Bridge) Done() <-chan struct{} {
	return b.done
}

// Connect will create a connection to the message broker with the provided context, by default
// it will attempt to connect at v3.1.1 and auto retry at v3.1 if that
// fails
func (b *Bridge) Connect(ctx context.Context) error {
	if len(b.m) == 0 {
		return errNoMetrics
	}
	t := b.client.Connect()
	return waitToken(ctx, t)
}

// IsConnected returns a bool signifying whether the bridge is connected or not.
func (b *Bridge) IsConnected() bool {
	return b.client.IsConnected()
}

// IsConnectionOpen return a bool signifying whether the bridge has an active
// connection to mqtt broker, i.e not in disconnected or reconnect mode
func (b *Bridge) IsConnectionOpen() bool {
	return b.client.IsConnectionOpen()
}

// Disconnect will end the connection with the server.
func (b *Bridge) Disconnect() {
	if err := b.publishStatus(nil, true); err != nil {
		log.WarnError("Unable to publish LWT on graceful disconnect", err)
	}
	b.client.Disconnect(500)
	defer func() {
		time.Sleep(time.Second)
		log.Info("Disconnected")
		if b.done != nil {
			close(b.done)
		}
	}()
	if b.ready == nil {
		return
	}
	<-b.ready
	b.cancel()
	b.wg.Wait()
	close(b.updates)
	if b.rediscover != nil {
		close(b.rediscover)
	}
}

// Discover publishes the discovery payload(s) for Home Assistant MQTT discovery after
// optionally waiting for a payload on the given wait topic. If path is a non-empty
// string, the previous discovery is loaded from the file at path for removing old
// components.
func (b *Bridge) Discover(ctx context.Context, path string) (err error) {
	var d, old *discovery.Discovery
	d, err = discovery.New(b.discoveryCfg)
	if err != nil {
		return err
	}
	for _, m := range b.m {
		if dd, ok := m.(discovery.Discoverer); ok {
			dd.Discover(d)
		}
	}
	if path != "" {
		old, err = discovery.Load(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	migrate := d.Diff(old)
	if err = d.Publish(ctx, b.client, migrate); err != nil {
		log.Error("Unable to perform discovery", err)
		return err
	}
	if path != "" {
		err = d.Write(path)
	}
	go func() {
		for m := range b.rediscover {
			dd, ok := m.(discovery.Discoverer)
			if !ok {
				continue
			}
			var cmps []string
			if d.Nodes != nil {
				node, ok := d.Nodes[m.Type()]
				if ok && node != nil {
					d.Nodes[m.Type()] = nil
					for _, c := range node {
						cmp, ok := d.Components[c]
						if !ok {
							continue
						}
						d.Components[c] = discovery.Component{
							discovery.Platform: cmp[discovery.Platform],
						}
					}
					cmps = node
				}
			}
			dd.Discover(d)
			if cmps != nil {
				node, ok := d.Nodes[m.Type()]
				if ok && len(cmps) > len(node) {
					d.Nodes[m.Type()] = cmps
				}
			}
			if err := d.Publish(ctx, b.client, false, m.Type()); err != nil {
				log.Error("Unable to perform discovery", err)
			}
		}
	}()
	log.Info("Discovery complete")
	return
}
