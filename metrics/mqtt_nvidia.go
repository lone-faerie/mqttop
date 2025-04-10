//go:build !nogpu

package metrics

import (
	"strconv"

	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/discovery/icon"
)

// GPU Discovery

// Discover implements [discovery.Discoverer]. Adds sensors for gpu usage,
// gpu power, gpu temperature, gpu memory usage, total gpu memory, free
// gpu memory, and used gpu memory.
func (g *NvidiaGPU) Discover(d *discovery.Discovery) {
	prefix := d.Origin.Name + "_gpu_" + strconv.Itoa(g.index)
	id := prefix
	avail := availabilityTemplate(g.Topic())

	var cmps []string

	if d.Nodes != nil {
		node, ok := d.Nodes[g.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 7)
		}

		cmps = node
	}

	if g.flags.Has(gpuUtilization) {
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 g.Name + " usage",
			discovery.Icon:                 icon.GPU,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           g.Topic(),
			discovery.ValueTemplate:        "{{ value_json.utilization.gpu }}",
			discovery.UnitOfMeasurement:    "%",
			discovery.UniqueID:             id,
		}
	}

	if g.flags.Has(gpuPower) {
		id = prefix + "_power"
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:               discovery.Sensor,
			discovery.Name:                   g.Name + " power",
			discovery.EntityCategory:         discovery.Diagnostic,
			discovery.DeviceClass:            "power",
			discovery.AvailabilityTopic:      d.AvailabilityTopic,
			discovery.AvailabilityTemplate:   avail,
			discovery.StateTopic:             g.Topic(),
			discovery.ValueTemplate:          "{{ value_json.power }}",
			discovery.UnitOfMeasurement:      "W",
			discovery.JSONAttributesTopic:    g.Topic(),
			discovery.JSONAttributesTemplate: "{{ {'max': value_json.maxPower} | tojson }}",
			discovery.UniqueID:               id,
		}
	}

	if g.flags.Has(gpuTemperature) {
		id = prefix + "_temperature"
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:               discovery.Sensor,
			discovery.Name:                   g.Name + " temperature",
			discovery.EntityCategory:         discovery.Diagnostic,
			discovery.DeviceClass:            "temperature",
			discovery.AvailabilityTopic:      d.AvailabilityTopic,
			discovery.AvailabilityTemplate:   avail,
			discovery.StateTopic:             g.Topic(),
			discovery.ValueTemplate:          "{{ value_json.temperature }}",
			discovery.UnitOfMeasurement:      "°C",
			discovery.JSONAttributesTopic:    g.Topic(),
			discovery.JSONAttributesTemplate: "{{ {'max': value_json.maxTemp} | tojson }}",
			discovery.UniqueID:               id,
		}
	}

	if g.flags.Has(gpuMemory | gpuMemoryV2 | gpuUtilization) {
		var template string

		if g.flags.Has(gpuUtilization) {
			template = "{{ value_json.utilization.memory }}"
		} else {
			template = "{{ 100 * value_json.memory.used / value_json.memory.total }}"
		}

		id = prefix + "_memory"
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 g.Name + " memory",
			discovery.Icon:                 icon.Memory,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           g.Topic(),
			discovery.ValueTemplate:        template,
			discovery.UnitOfMeasurement:    "%",
			discovery.UniqueID:             id,
		}

		if g.flags.Has(gpuMemory | gpuMemoryV2) {
			id = prefix + "_memory_total"
			if cmps != nil {
				cmps = append(cmps, id)
			}

			d.Components[id] = discovery.Component{
				discovery.Platform:             discovery.Sensor,
				discovery.Name:                 g.Name + " memory total",
				discovery.Icon:                 icon.Memory,
				discovery.EntityCategory:       discovery.Diagnostic,
				discovery.DeviceClass:          "data_size",
				discovery.AvailabilityTopic:    d.AvailabilityTopic,
				discovery.AvailabilityTemplate: avail,
				discovery.StateTopic:           g.Topic(),
				discovery.ValueTemplate:        "{{ value_json.memory.total }}",
				discovery.UnitOfMeasurement:    g.memSize,
				discovery.UniqueID:             id,
				discovery.EnabledByDefault:     false,
			}

			id = prefix + "_memory_free"
			if cmps != nil {
				cmps = append(cmps, id)
			}

			d.Components[id] = discovery.Component{
				discovery.Platform:             discovery.Sensor,
				discovery.Name:                 g.Name + " memory free",
				discovery.Icon:                 icon.Memory,
				discovery.EntityCategory:       discovery.Diagnostic,
				discovery.DeviceClass:          "data_size",
				discovery.AvailabilityTopic:    d.AvailabilityTopic,
				discovery.AvailabilityTemplate: avail,
				discovery.StateTopic:           g.Topic(),
				discovery.ValueTemplate:        "{{ value_json.memory.free }}",
				discovery.UnitOfMeasurement:    g.memSize,
				discovery.UniqueID:             id,
				discovery.EnabledByDefault:     false,
			}

			id = prefix + "_memory_used"
			if cmps != nil {
				cmps = append(cmps, id)
			}

			d.Components[id] = discovery.Component{
				discovery.Platform:             discovery.Sensor,
				discovery.Name:                 g.Name + " memory used",
				discovery.Icon:                 icon.Memory,
				discovery.EntityCategory:       discovery.Diagnostic,
				discovery.DeviceClass:          "data_size",
				discovery.AvailabilityTopic:    d.AvailabilityTopic,
				discovery.AvailabilityTemplate: avail,
				discovery.StateTopic:           g.Topic(),
				discovery.ValueTemplate:        "{{ value_json.memory.used }}",
				discovery.UnitOfMeasurement:    g.memSize,
				discovery.UniqueID:             id,
				discovery.EnabledByDefault:     false,
			}
		}
	}

	if g.flags.Has(gpuThroughput) {
		id = prefix + "_rx"
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 g.Name + " rx",
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_rate",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           g.Topic(),
			discovery.ValueTemplate:        "{{ value_json.rx }}",
			discovery.UnitOfMeasurement:    "kB/s",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}

		id = prefix + "_tx"
		if cmps != nil {
			cmps = append(cmps, id)
		}

		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 g.Name + " tx",
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_rate",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           g.Topic(),
			discovery.ValueTemplate:        "{{ value_json.tx }}",
			discovery.UnitOfMeasurement:    "kB/s",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
	}

	if cmps != nil {
		d.Nodes[g.Type()] = cmps
	}
}
