package discovery

import (
	"errors"
	"strings"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/internal/build"
)

const (
	BinarySensor = "binary_sensor"
	Button       = "button"
	Sensor       = "sensor"
	Switch       = "switch"
)

const (
	Diagnostic = "diagnostic"
)

type Component map[Option]any

type Discoverer interface {
	Discover(*Discovery)
}

type Discovery struct {
	Origin     *Origin              `json:"o"`
	Device     *Device              `json:"dev"`
	Components map[string]Component `json:"cmps"`

	cfg *config.DiscoveryConfig

	AvailabilityTopic string `json:"-"`
	ObjectID          string `json:"-"`
	NodeID            string `json:"-"`
}

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

func (d *Discovery) Topic(prefix string) (string, error) {
	elems := []string{prefix, "device", d.NodeID, d.ObjectID, "config"}
	return strings.Join(elems, "/"), nil
}

func (d *Discovery) SetAvailability(avail Component) {
	for cmp := range d.Components {
		d.Components[cmp][Availability] = avail
	}
}

type Origin struct {
	Name       string `json:"name"`
	SWVersion  string `json:"sw,omitempty"`
	SupportURL string `json:"url,omitempty"`
}

func NewOrigin() *Origin {
	o := &Origin{
		Name:       "mqttop",
		SWVersion:  build.Version(),
		SupportURL: "https://" + build.Package(),
	}
	return o
}
