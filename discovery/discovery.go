// Package discovery provides structures to support Home Assistant MQTT Discovery/
package discovery

import (
	"errors"
	"strings"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/build"
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
	Discover(*Discovery)
}

// Discovery is the struct that is encoded into the device discovery payload.
type Discovery struct {
	Origin     *Origin              `json:"o"`
	Device     *Device              `json:"dev"`
	Components map[string]Component `json:"cmps"`

	cfg *config.DiscoveryConfig

	AvailabilityTopic string `json:"-"`
	ObjectID          string `json:"-"`
	NodeID            string `json:"-"`
}

// New returns a new Discovery struct initialized from the provided config and components.
func New(cfg *config.DiscoveryConfig, cmps ...Discoverer) (*Discovery, error) {
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
		Components:        make(map[string]Component, len(cmps)),
		NodeID:            cfg.NodeID,
		AvailabilityTopic: cfg.Availability,
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
	for i := range cmps {
		cmps[i].Discover(d)
	}
	return d, nil
}

// Topic returns the topic to publish the discovery payload to using the provided prefix.
func (d *Discovery) Topic(prefix string) (string, error) {
	elems := []string{prefix, "device", d.NodeID, d.ObjectID, "config"}
	return strings.Join(elems, "/"), nil
}

// SetAvailability sets the availability of all components to the one provided.
func (d *Discovery) SetAvailability(avail Component) {
	for cmp := range d.Components {
		d.Components[cmp][Availability] = avail
	}
}

// Origin implements the origin mapping for the discovery payload. This provides context to
// Home Assistant on the origin of the components.
type Origin struct {
	Name       string `json:"name"`
	SWVersion  string `json:"sw,omitempty"`
	SupportURL string `json:"url,omitempty"`
}

// NewOrigin returns the default Origin with the following values:
// - Name: "mqttop"
// - SWVersion: [build.Version()]
// - SupportURL: "https://github.com/lone-faerie/mqttop"
func NewOrigin() *Origin {
	o := &Origin{
		Name:       "mqttop",
		SWVersion:  build.Version(),
		SupportURL: "https://" + build.Package(),
	}
	return o
}
