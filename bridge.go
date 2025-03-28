package mqttop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

type stateMap struct {
	m  map[string]bool
	mu sync.Mutex
}

func (m *stateMap) Set(key string, state bool) {
	m.mu.Lock()
	m.m[key] = state
	m.mu.Unlock()
}

func (m *stateMap) Delete(key string) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

func (m *stateMap) MarshalJSON() ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return json.Marshal(m.m)
}

var logOnce sync.Once

var errNoMetrics = errors.New("no metrics")

// Bridge is the mqtt client that bridges metrics to the mqtt broker.
type Bridge struct {
	client mqtt.Client

	discoveryCfg *config.DiscoveryConfig
	m            []metrics.Metric
	states       stateMap

	updates chan metrics.Metric
	once    sync.Once

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	ready  chan struct{}
}

// New returns a new Bridge with the provided config and a [mqtt.Client] derived from the config.
// The bridge must have [Bridge.Connect] and [Bridge.Ready] called on it before it may be used.
// This follows the convention of [mqtt.NewClient] as well as waiting for metrics to be ready.
func New(cfg *config.Config) *Bridge {
	opts := cfg.MQTT.ClientOptions().SetWill(
		cfg.MQTT.BirthWillTopic, "offline", 1, true,
	)
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
	return &Bridge{
		client:       c,
		m:            metrics.New(cfg),
		discoveryCfg: &cfg.Discovery,
	}
}

func (b *Bridge) handleMetric(i int, m metrics.Metric) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		switch {
		case strings.HasSuffix(msg.Topic(), "update"):
			go func() {
				if err := m.Update(); err == nil {
					b.updates <- m
				}
			}()
		case strings.HasSuffix(msg.Topic(), "stop"):
			go m.Stop()
		}
	}
}

func (b *Bridge) publishBirthOrWill(ctx context.Context, isBirth bool) (err error) {
	var (
		data []byte
		opts = b.client.OptionsReader()
	)
	if ctx == nil {
		ctx = context.Background()
	}
	if isBirth {
		data, err = json.Marshal(&b.states)
		if err != nil {
			return
		}
	} else {
		data = opts.WillPayload()
	}
	t := b.client.Publish(opts.WillTopic(), opts.WillQos(), opts.WillRetained(), data)
	select {
	case <-ctx.Done():
		return
	case <-t.Done():
	}
	return t.Error()
}

func (b *Bridge) publishUpdates(ctx context.Context) {
	var done <-chan struct{}
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-b.updates:
			if !ok {
				return
			}
			data, _ := m.AppendText(nil)
			t := b.client.Publish(m.Topic(), 0, false, data)
			done = t.Done()
		case <-done:
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
		b.start(ctx)
	})
}

func (b *Bridge) startMetric(ctx context.Context, i int, m metrics.Metric) (done bool) {
	if m.Topic() == "" {
		return
	}
	if err := m.Start(ctx); err != nil {
		log.Error("Error starting "+m.Type(), err)
		b.states.Set(m.Topic(), false)
		return
	}
	b.states.Set(m.Topic(), true)
	t := b.client.SubscribeMultiple(metricTopics(m), b.handleMetric(i, m))
	select {
	case <-ctx.Done():
		return true
	case <-t.Done():
	}
	if err := t.Error(); err != nil {
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
		log.Info(metric.Type() + " started")
		for err := range ch {
			if err == nil {
				b.updates <- metric
			} else if err != metrics.ErrNoChange {
				log.Warn("Error updating metric", "metric", metric.Type(), "err", err)
			}
		}
		log.Info(metric.Type() + " done")
	}(i, m)
	return
}

// start starts listening to the metrics.
func (b *Bridge) start(ctx context.Context) {
	b.ready = make(chan struct{})
	b.updates = make(chan metrics.Metric)
	b.states.m = make(map[string]bool, len(b.m))
	ctx, b.cancel = context.WithCancel(ctx)
	go func() {
		defer close(b.ready)
		for i, m := range b.m {
			if b.startMetric(ctx, i, m) {
				break
			}
		}
		if err := b.publishBirthOrWill(ctx, true); err != nil {
			log.Error("Unable to publish birth message", err)
		}
		go b.publishUpdates(ctx)
	}()
	return
}

// Ready returns a channel that can be used to wait until all metrics have been started.
func (b *Bridge) Ready() <-chan struct{} {
	return b.ready
}

// Connect will create a connection to the message broker with the provided context, by default
// it will attempt to connect at v3.1.1 and auto retry at v3.1 if that
// fails
func (b *Bridge) Connect(ctx context.Context) error {
	if len(b.m) == 0 {
		return errNoMetrics
	}
	t := b.client.Connect()
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	return t.Error()
}

// Disconnect will end the connection with the server.
func (b *Bridge) Disconnect() {
	if err := b.publishBirthOrWill(nil, false); err != nil {
		log.Warn("Unable to publish LWT on graceful disconnect", err)
	}
	b.client.Disconnect(500)
	if b.ready != nil {
		<-b.ready
	}
	b.cancel()
	b.wg.Wait()
	close(b.updates)
	time.Sleep(time.Second)
	log.Debug("Disconnected")
}

func (b *Bridge) waitDiscover(ctx context.Context) error {
	if b.discoveryCfg.WaitTopic == "" {
		log.Debug("Not waiting before discovery")
		return nil
	}
	ch := make(chan error)
	defer close(ch)
	handler := func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		if string(msg.Payload()) == b.discoveryCfg.WaitPayload {
			t := b.client.Unsubscribe(b.discoveryCfg.WaitTopic)
			select {
			case <-ctx.Done():
			case <-t.Done():
			}
			select {
			case ch <- t.Error():
			default:
			}
		}
	}
	t := b.client.Subscribe(b.discoveryCfg.WaitTopic, 0, handler)
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	if err := t.Error(); err != nil {
		return err
	}
	return <-ch
}

// Discover publishes the discovery payload for Home Assistant MQTT discovery after
// optionally waiting for a payload on the given wait topic.
func (b *Bridge) Discover(ctx context.Context) error {
	if err := b.waitDiscover(ctx); err != nil {
		log.Warn("Could not wait for discovery", err)
		return err
	}
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	disc, err := discovery.New(b.discoveryCfg)
	if err != nil {
		log.Error("Unable to get discovery", err)
		return err
	}
	for _, metric := range b.m {
		if d, ok := metric.(discovery.Discoverer); ok {
			d.Discover(disc)
		}
	}
	pay, err := json.Marshal(disc)
	if err != nil {
		log.Error("Unable to marshal discovery payload", err)
		return err
	}
	topic, err := disc.Topic(b.discoveryCfg.Prefix)
	if err != nil {
		log.Error("Unable to get discovery topic", err)
		return err
	}
	t := b.client.Publish(topic, b.discoveryCfg.QoS, b.discoveryCfg.Retained, pay)
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	log.Info("discovery finished")
	if err = t.Error(); err != nil {
		log.Warn("Unable to publish discovery", err)
	}
	return err
}
