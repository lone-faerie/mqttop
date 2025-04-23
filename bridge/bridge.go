package bridge

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
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

// Bridge is the mqtt client that bridges metrics to the mqtt broker.
type Bridge struct {
	client mqtt.Client

	baseTopic string
	discovery *discovery.Discovery
	migrate   bool
	metrics   []metrics.Metric
	states    sync.Map

	updates    chan metrics.Metric
	rediscover chan metrics.Metric

	ready chan struct{}
	done  chan struct{}
	err   error

	mu     sync.Mutex
	wg     sync.WaitGroup
	once   sync.Once
	cancel context.CancelFunc
}

var noopLogger = mqtt.NOOPLogger{}

// New returns a new Bridge with the givenn config and options. The config will be used to fill
// in any necessary values not provided by the options. The bridge must have [Bridge.Connect]
// and [Bridge.Ready] called on it before it may be used. This follows the convention of
// [mqtt.NewClient] as well as waiting for metrics to be ready.
func New(cfg *config.Config, opts ...Option) *Bridge {
	b := &Bridge{}

	for _, opt := range opts {
		opt(b)
	}

	if b.client == nil {
		opts := cfg.MQTT.ClientOptions()
		b.client = mqtt.NewClient(opts)
	}

	if len(b.metrics) == 0 {
		b.metrics = metrics.New(cfg)
	}

	if b.discovery == nil && cfg.Discovery.Enabled {
		d, err := discovery.New(&cfg.Discovery)
		if err != nil {
			log.Error("Unable to get discovery", err)
		} else {
			b.discovery = d
		}
	}

	if cfg.MQTT.LogLevel < log.LevelDisabled && mqtt.ERROR != noopLogger {
		WithLogLevel(cfg.MQTT.LogLevel)(b)
	}

	if b.baseTopic == "" {
		if cfg.BaseTopic != "" {
			b.baseTopic = cfg.BaseTopic
		} else {
			b.baseTopic = "mqttop"
		}
	}

	return b
}

func (b *Bridge) AddMetric(ctx context.Context, m metrics.Metric) {
	var done <-chan struct{}

	if ctx != nil {
		done = ctx.Done()
	}

	select {
	case <-done:
		return
	case <-b.done:
		return
	case <-b.ready:
		b.mu.Lock()

		i := len(b.metrics)
		b.metrics = append(b.metrics, m)

		b.mu.Unlock()
		b.startMetric(ctx, i, m, true)
	default:
		b.metrics = append(b.metrics, m)
	}
}

// waitToken waits for the first of ctx.Done() or t.Done() and returns t.Error(), or nil if
// ctx.Done() finished first.
func waitToken(ctx context.Context, t mqtt.Token) error {
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}

	return t.Error()
}

// ctxDone indicates whether the given context has been canceled.
func ctxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// maybeSend sends t on ch, unless the given context is cancelled before it can send.
// maybeSend returns true if t was sent and false if the context was canceled.
func maybeSend[T any](ctx context.Context, ch chan<- T, t T) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- t:
		return true
	}
}

// loopMetric is the event loop for the given metric and listens for updates on its [metrics.Metric.Updated] channel.
func (b *Bridge) loopMetric(ctx context.Context, i int, m metrics.Metric) {
	defer func() {
		m.Stop()

		b.mu.Lock()
		b.metrics[i] = nil
		b.mu.Unlock()

		b.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-m.Updated():
			updated := b.updateState(ctx, m, err)

			switch err {
			case nil:
				maybeSend(ctx, b.updates, m)
			case metrics.ErrNoChange:
				if updated {
					maybeSend(ctx, b.updates, m)
				}
			case metrics.ErrRescanned:
				if b.rediscover != nil {
					maybeSend(ctx, b.rediscover, m)
				}
			default:
				log.WarnError("Error updating "+m.Type(), err)
			}
		}
	}
}

// nilToken implements [mqtt.Token] with a nil channel.
type nilToken struct{}

func (nilToken) Wait() bool                       { return true }
func (nilToken) WaitTimeout(_ time.Duration) bool { return true }
func (nilToken) Done() <-chan struct{}            { return nil }
func (nilToken) Error() error                     { return nil }

// loop is the event loop for the bridge and publishes any metrics received on the updates channel.
func (b *Bridge) loop(ctx context.Context) {
	defer func() {
		if b.client.IsConnected() || b.client.IsConnectionOpen() {
			t := b.publishStates(true)
			t.Wait()

			b.client.Disconnect(500)
		}

		close(b.updates)

		if b.rediscover != nil {
			close(b.rediscover)
		}

		b.wg.Wait()

		close(b.done)
	}()

	var t mqtt.Token = nilToken{}

	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-b.updates:
			if !ok {
				return
			}

			data, err := m.AppendText(nil)
			if err != nil {
				log.WarnError("Unable to marshal "+m.Type(), err)
				break
			}

			t = b.client.Publish(m.Topic(), 0, false, data)
		case m, ok := <-b.rediscover:
			if !ok {
				return
			}

			err := b.publishRediscovery(ctx, m)
			if err != nil {
				log.WarnError("Unable to publish discovery", err)
			}
		case <-t.Done():
			if err := t.Error(); err != nil {
				log.WarnError("Unable to publish update", err)
			}

			t = nilToken{}
		}
	}
}

// updateState updates the state for the given metric in the bridge's states map. If the state changed,
// updateState returns true and publishes the updated states to the LWT topic.
func (b *Bridge) updateState(ctx context.Context, m metrics.Metric, err error) (updated bool) {
	key := m.Topic()
	state := err == nil || err == metrics.ErrNoChange || err == metrics.ErrRescanned

	if updated = b.states.CompareAndSwap(key, !state, state); !updated {
		return
	}

	log.Debug("State changed", "topic", key, "from", !state, "to", state)

	t := b.publishStates(false)
	if err := waitToken(ctx, t); err != nil {
		log.WarnError("Unable to publish states", err)
	}

	return
}

func handleUpdatePayload(m metrics.Metric, payload []byte) error {
	if len(payload) == 0 {
		return nil
	}

	var mm map[string]string

	if err := json.Unmarshal(payload, &mm); err != nil {
		return err
	}

	if interval, ok := mm["interval"]; ok {
		d, err := time.ParseDuration(interval)
		if err == nil {
			m.SetInterval(d)
		}
	}

	if mode, ok := mm["selection_mode"]; ok {
		if c, ok := m.(*metrics.CPU); ok {
			c.SetSelectionMode(mode)
		}
	}

	return nil
}

// metricHandler returns a [mqtt.MessageHandler] for the given metric that handles the "/update" and "/stop"
// topics of the metric.
func (b *Bridge) metricHandler(ctx context.Context, i int, m metrics.Metric) mqtt.MessageHandler {
	return func(_ mqtt.Client, msg mqtt.Message) {
		switch {
		case strings.HasSuffix(msg.Topic(), "/update"):
			go func(msg mqtt.Message) {
				handleUpdatePayload(m, msg.Payload())

				if err := m.Update(); err == nil {
					maybeSend(ctx, b.updates, m)
				}
			}(msg)
		case strings.HasSuffix(msg.Topic(), "/stop"):
			go m.Stop()
		}
	}
}

// startMetric initializes the given metric and starts its event loop.
func (b *Bridge) startMetric(ctx context.Context, i int, m metrics.Metric, discover bool) {
	if m.Topic() == "" {
		log.Debug("No topic, skipping", "metric", m.Type())
		return
	}

	if err := m.Start(ctx); err != nil {
		log.Error("Could not start "+m.Type(), err)
		b.states.Store(m.Topic(), false)

		return
	}

	b.states.Store(m.Topic(), true)

	t := b.client.SubscribeMultiple(map[string]byte{
		m.Topic() + "/update": 0,
		m.Topic() + "/stop":   0,
	}, b.metricHandler(ctx, i, m))
	if err := waitToken(ctx, t); err != nil {
		log.Error("Could not subscribe to "+m.Topic(), err)
		m.Stop()

		return
	}

	b.wg.Add(1)

	go b.loopMetric(ctx, i, m)

	if discover && b.rediscover != nil {
		maybeSend(ctx, b.rediscover, m)
	}
}

// start starts the bridge's metrics and the bridge's event loop.
func (b *Bridge) start(ctx context.Context) {
	defer func() {
		select {
		case <-ctx.Done():
		default:
			close(b.ready)
		}
	}()

	for i, m := range b.metrics {
		b.startMetric(ctx, i, m, false)

		if ctxDone(ctx) {
			return
		}
	}

	t := b.publishStates(false)
	if err := waitToken(ctx, t); err != nil {
		b.err = err
	}

	t = b.client.Subscribe(b.baseTopic+"/bridge/stop", 0, func(_ mqtt.Client, _ mqtt.Message) {
		go b.Stop()
	})
	if err := waitToken(ctx, t); err != nil && b.err == nil {
		b.err = err
	}

	t = b.client.Subscribe(b.baseTopic+"/bridge/update", 0, func(_ mqtt.Client, _ mqtt.Message) {
		go b.update(ctx)
	})
	if err := waitToken(ctx, t); err != nil && b.err == nil {
		b.err = err
	}

	if b.discovery != nil {
		if err := b.discover(ctx); err != nil && b.err == nil {
			b.err = err
		}
	}

	b.done = make(chan struct{})

	go b.loop(ctx)
}

func (b *Bridge) Start(ctx context.Context) error {
	if len(b.metrics) == 0 {
		return errors.New("no metrics")
	}

	t := b.client.Connect()
	if err := waitToken(ctx, t); err != nil {
		return err
	}

	b.once.Do(func() {
		b.ready = make(chan struct{})
		b.updates = make(chan metrics.Metric)

		if b.discovery != nil {
			b.rediscover = make(chan metrics.Metric)
		}

		ctx, b.cancel = context.WithCancel(ctx)

		go b.start(ctx)
	})

	return nil
}

func (b *Bridge) Stop() {
	log.Debug("Stopping bridge")

	if b.ready == nil {
		return
	}

	<-b.ready
	b.cancel()

	if b.done != nil {
		<-b.done
	}
}

func (b *Bridge) Ready() <-chan struct{} {
	return b.ready
}

func (b *Bridge) Done() <-chan struct{} {
	return b.done
}

func (b *Bridge) Error() error {
	return b.err
}

func (b *Bridge) update(ctx context.Context) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var wg sync.WaitGroup

	for _, m := range b.metrics {
		if m == nil {
			continue
		}

		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		go func(m metrics.Metric) {
			defer wg.Done()

			err := m.Update()
			b.updateState(ctx, m, err)

			if err != nil && err != metrics.ErrNoChange {
				log.WarnError("Error updating "+m.Type(), err)
				return
			}

			maybeSend(ctx, b.updates, m)
		}(m)
	}

	wg.Wait()
}

// publishStates publishes the bridge's states map to the LWT topic. If lwt is true, publishState
// publishes the client's LWT payload instead.
func (b *Bridge) publishStates(lwt bool) mqtt.Token {
	var (
		payload []byte
		opts    = b.client.OptionsReader()
	)

	if lwt {
		payload = opts.WillPayload()
	} else {
		payload = []byte{'{'}
		first := true

		b.states.Range(func(k, v any) bool {
			if !first {
				payload = append(payload, ',')
			}

			payload = strconv.AppendQuote(payload, k.(string))
			payload = append(payload, ':')
			payload = strconv.AppendBool(payload, v.(bool))

			first = false

			return true
		})

		payload = append(payload, '}')
	}

	return b.client.Publish(opts.WillTopic(), opts.WillQos(), opts.WillRetained(), payload)
}

func (b *Bridge) publishRediscovery(ctx context.Context, m metrics.Metric) error {
	dd, ok := m.(discovery.Discoverer)
	if !ok || b.discovery == nil {
		return nil
	}

	var cmps []string

	if b.discovery.Nodes != nil {
		node, ok := b.discovery.Nodes[m.Type()]
		if ok && node != nil {
			b.discovery.Nodes[m.Type()] = nil

			for _, c := range node {
				cmp, ok := b.discovery.Components[c]
				if !ok {
					continue
				}

				b.discovery.Components[c] = discovery.Component{
					discovery.Platform: cmp[discovery.Platform],
				}
			}

			cmps = node
		}
	}

	dd.Discover(b.discovery)

	if cmps != nil {
		node, ok := b.discovery.Nodes[m.Type()]

		if ok && len(cmps) > len(node) {
			slices.Sort(cmps)
			b.discovery.Nodes[m.Type()] = slices.Compact(cmps)
		}
	}

	return b.discovery.Publish(ctx, b.client, false, m.Type())
}

func (b *Bridge) discover(ctx context.Context) error {
	b.Discover(b.discovery)

	if err := b.discovery.Publish(ctx, b.client, b.migrate); err != nil {
		return err
	}

	return b.discovery.SubscribeFunc(ctx, b.client, func(ctx context.Context) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
		b.update(ctx)
	})
}

func (b *Bridge) Discover(d *discovery.Discovery) {
	var cmps []string

	if d.Nodes != nil {
		node, ok := d.Nodes["bridge"]
		if !ok || node == nil {
			node = make([]string, 0, 2)
		}

		cmps = node
	}

	id := d.Origin.Name + "_update"
	if cmps != nil {
		cmps = append(cmps, id)
	}

	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Button,
		discovery.Name:                 "Update",
		discovery.DeviceClass:          "restart",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: "{{ iif(value == 'offline', value, 'online') }}",
		discovery.CommandTopic:         b.baseTopic + "/bridge/update",
		discovery.UniqueID:             id,
	}

	if cmps != nil {
		d.Nodes["bridge"] = cmps
	}
}
