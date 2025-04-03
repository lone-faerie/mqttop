// Package discovery provides structures to support Home Assistant MQTT Discovery.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/eclipse/paho.mqtt.golang"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
)

// Home Assistant entity platforms
const (
	BinarySensor = "binary_sensor"
	Button       = "button"
	Sensor       = "sensor"
	Switch       = "switch"
)

// Home Assitant entity categories
const (
	Diagnostic = "diagnostic"
)

type Component map[Option]any

// Discoverer is the interface that is implemented by a value to add it to the discovery
// payload. Implementations should add each of its Components to the provided Discovery's Components field using unique keys.
type Discoverer interface {
	Discover(d *Discovery)
}

// Discovery is the struct that is encoded into the device discovery payload.
type Discovery struct {
	Origin     *Origin              `json:"o"`
	Device     *Device              `json:"dev"`
	Components map[string]Component `json:"cmps"`

	cfg *config.DiscoveryConfig

	AvailabilityTopic string              `json:"-"`
	ObjectID          string              `json:"-"`
	NodeID            string              `json:"-"`
	Nodes             map[string][]string `json:"_nodes,omitempty"`
	Method            string              `json:"_method,omitempty"`
}

// Load returns the decoded value of a discovery payload at the file path.
func Load(path string) (*Discovery, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	d := &Discovery{}
	if err := json.NewDecoder(f).Decode(d); err != nil {
		return nil, err
	}
	return d, nil
}

// New returns a new Discovery struct initialized from the provided config and components.
func New(cfg *config.DiscoveryConfig) (*Discovery, error) {
	dev, err := NewDevice()
	if err != nil {
		return nil, err
	}
	switch cfg.DeviceName {
	case "", "hostname":
	default:
		dev.Name = cfg.DeviceName
	}
	if dev.Name == "" {
		dev.Name = "Mqttop"
	}

	d := &Discovery{
		Origin:            NewOrigin(),
		Device:            dev,
		Components:        make(map[string]Component),
		NodeID:            cfg.NodeID,
		AvailabilityTopic: cfg.Availability,
		cfg:               cfg,
		Method:            cfg.Method,
	}
	if d.Method == "nodes" || d.Method == "metrics" {
		d.Nodes = make(map[string][]string)
	}
	if d.NodeID == "" {
		d.NodeID = "mqttop"
	}
	switch {
	case len(dev.Identifiers) > 0:
		d.ObjectID = strings.Join(dev.Identifiers, "_")
	case len(dev.Connections) > 0:
		for i := range dev.Connections {
			if i > 0 {
				d.ObjectID += "_"
			}
			d.ObjectID += dev.Connections[i][1]
		}
	default:
		return nil, errors.New("No object id")
	}
	return d, nil
}

// Topic returns the topic to publish the discovery payload to using the provided prefix.
func (d *Discovery) Topic(prefix, component, nodeID, objectID string) (string, error) {
	if objectID == "" {
		objectID = d.ObjectID
	}
	var elems []string
	if nodeID != "" {
		elems = []string{prefix, component, nodeID, objectID, "config"}
	} else {
		elems = []string{prefix, component, objectID, "config"}
	}
	return strings.Join(elems, "/"), nil
}

// SetAvailability sets the availability of all components to the one provided.
func (d *Discovery) SetAvailability(avail Component) {
	for cmp := range d.Components {
		d.Components[cmp][Availability] = avail
	}
}

// Write writes the json-encoded value of d to path.
func (d *Discovery) Write(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	f.Truncate(0)
	f.Seek(0, 0)
	e := json.NewEncoder(f)
	e.SetIndent("", "  ")
	return e.Encode(d)
}

// Wait blocks until the given payload is received on the wait topic, if defined,
// otherwise Wait returns immediately.
func (d *Discovery) Wait(ctx context.Context, c mqtt.Client) error {
	if d.cfg.WaitTopic == "" {
		return nil
	}
	ch := make(chan error)
	defer close(ch)
	t := c.Subscribe(d.cfg.WaitTopic, 0, func(_ mqtt.Client, msg mqtt.Message) {
		msg.Ack()
		if d.cfg.WaitPayload == "" || string(msg.Payload()) == d.cfg.WaitPayload {
			t := c.Unsubscribe(d.cfg.WaitTopic)
			select {
			case <-ctx.Done():
			case <-t.Done():
			}
			select {
			case ch <- t.Error():
			default:
			}
		}
	})
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

func (d *Discovery) publishDevice(ctx context.Context, c mqtt.Client, migrate bool) error {
	nodes := d.Nodes
	d.Nodes = nil
	defer func() {
		d.Nodes = nodes
	}()
	if migrate {
		if err := d.Migrate(ctx, c); err != nil {
			return err
		}
	}
	if err := d.publishDeviceNode(ctx, c, d.NodeID); err != nil {
		return err
	}
	if migrate {
		return d.removeComponents(ctx, c)
	}
	return nil
}

func (d *Discovery) publishDeviceNode(ctx context.Context, c mqtt.Client, nodeID string) error {
	payload, err := json.Marshal(d)
	if err != nil {
		return err
	}
	topic, err := d.Topic(d.cfg.Prefix, "device", nodeID, d.ObjectID)
	if err != nil {
		return err
	}
	t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, payload)
	select {
	case <-ctx.Done():
		return nil
	case <-t.Done():
	}
	if err := t.Error(); err != nil {
		return err
	}
	return nil
}

func (d *Discovery) publishComponents(ctx context.Context, c mqtt.Client, migrate bool, components ...string) (err error) {
	nodes := d.Nodes
	d.Nodes = nil
	defer func() {
		d.Nodes = nodes
	}()
	if migrate {
		if err = d.Rollback(ctx, c); err != nil {
			return
		}
	}
	var payload []byte
	for name, cmp := range d.Components {
		if len(components) > 0 && !slices.Contains(components, name) {
			continue
		}
		platform := cmp[Platform].(string)
		if len(cmp) == 1 {
			payload = []byte{}
		} else {
			delete(cmp, Platform)
			cmp[optOrigin] = d.Origin
			cmp[optDevice] = d.Device
			payload, err = json.Marshal(cmp)
			if err != nil {
				return err
			}
			cmp[Platform] = platform
			delete(cmp, optOrigin)
			delete(cmp, optDevice)
		}
		topic, err := d.Topic(d.cfg.Prefix, platform, d.NodeID, name)
		if err != nil {
			return err
		}
		t := c.Publish(topic, d.cfg.QoS, d.cfg.Retained, payload)
		select {
		case <-ctx.Done():
			return nil
		case <-t.Done():
		}
		if err := t.Error(); err != nil {
			return err
		}
	}
	if migrate {
		return d.removeDevice(ctx, c)
	}
	return nil
}

func (d *Discovery) publishNodes(ctx context.Context, c mqtt.Client, migrate bool, nodes ...string) error {
	dNodes := d.Nodes
	d.Nodes = nil
	defer func() {
		d.Nodes = dNodes
	}()
	nodeD := Discovery{
		Origin:     d.Origin,
		Device:     d.Device,
		Components: make(map[string]Component),
		ObjectID:   d.ObjectID,
		cfg:        d.cfg,
	}
	var it iter.Seq[string]
	if len(nodes) > 0 {
		it = slices.Values(nodes)
	} else {
		it = maps.Keys(dNodes)
	}
	for node := range it {
		cmps, ok := dNodes[node]
		if !ok || len(cmps) == 0 {
			continue
		}
		clear(nodeD.Components)
		for _, c := range cmps {
			cmp, ok := d.Components[c]
			if ok {
				nodeD.Components[c] = cmp
			}
		}
		if len(nodeD.Components) == 0 {
			continue
		}
		if err := nodeD.publishDeviceNode(ctx, c, d.NodeID+"_"+node); err != nil {
			return err
		}
	}
	return nil
}

// Publish publishes the discovery payload. If migrate is true, Publish migrates the discovery payload
// either from a device discovery to individual component discoveries, or from individual component
// discoveries to a device discovery.
func (d *Discovery) Publish(ctx context.Context, c mqtt.Client, migrate bool, args ...string) (err error) {
	if err = d.Wait(ctx, c); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return nil
	default:
	}
	method := d.Method
	d.Method = ""
	defer func() {
		d.Method = method
	}()
	switch method {
	case "", "device":
		log.Debug("Publishing discovery", "method", "device")
		err = d.publishDevice(ctx, c, migrate)
	case "components":
		log.Debug("Publishing discovery", "method", "components")
		err = d.publishComponents(ctx, c, migrate, args...)
	case "nodes", "metrics":
		log.Debug("Publishing discovery", "method", "nodes")
		err = d.publishNodes(ctx, c, migrate, args...)
	}
	if err != nil {
		log.Error("Unsuccessful discovery", err)
	}
	return
}

func shouldMigrate(method, old string) bool {
	switch old {
	case "", "device":
		return method == "components"
	case "components":
		return method == "" || method == "device"
	}
	return false
}

// Diff adds an empty component to d for each component in old that
// isn't already in d. Diff returns true if d should be migrated.
func (d *Discovery) Diff(old *Discovery) bool {
	if old == nil {
		return false
	}
	for name, cmp := range old.Components {
		if _, ok := d.Components[name]; ok || len(cmp) <= 1 {
			continue
		}
		d.Components[name] = Component{
			Platform: cmp[Platform],
		}
	}
	return shouldMigrate(d.Method, old.Method)
}
