package metrics

import (
	"fmt"
	"strconv"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/discovery/icon"
	"github.com/lone-faerie/mqttop/internal/byteutil"
)

type Publisher interface {
	Publish(mqtt.Client) mqtt.Token
}

type Subscriber interface {
	Subscribe(mqtt.Client) mqtt.Token
}

type MQTTMetric interface {
	Metric
	Publisher
	Subscriber
}

func availabilityTemplate(topic string) string {
	return fmt.Sprintf(
		"{{ iif(value_json[%q]|default, 'online', 'offline') if value_json is defined else value }}",
		topic,
	)
}

// Battery Discovery
func (b *Battery) Discover(d *discovery.Discovery) {
	id := d.Origin.Name + "_battery_state"
	avail := availabilityTemplate(b.Topic())
	d.Components[id] = discovery.Component{
		discovery.Platform:               discovery.Sensor,
		discovery.Name:                   "Battery state",
		discovery.Icon:                   icon.Battery,
		discovery.EntityCategory:         discovery.Diagnostic,
		discovery.DeviceClass:            "enum",
		discovery.AvailabilityTopic:      d.AvailabilityTopic,
		discovery.AvailabilityTemplate:   avail,
		discovery.StateTopic:             b.Topic(),
		discovery.ValueTemplate:          "{{ value_json.status }}",
		discovery.JSONAttributesTopic:    b.Topic(),
		discovery.JSONAttributesTemplate: "{{ {'kind': value_json.kind } | tojson }}",
		discovery.Options: []string{
			"unknown", "charging", "discharging", "not charging", "full",
		},
		discovery.UniqueID: id,
	}
	id = d.Origin.Name + "_battery_charging"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.BinarySensor,
		discovery.Name:                 "Battery charging",
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "battery_charging",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           b.Topic(),
		discovery.ValueTemplate:        "{{ iif(value_json.status == 'charging', 'ON', 'OFF') }}",
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	if b.hasCapacity() {
		id = d.Origin.Name + "_battery_level"
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 "Battery level",
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "battery",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           b.Topic(),
			discovery.ValueTemplate:        "{{ value_json.capacity }}",
			discovery.UnitOfMeasurement:    "%",
			discovery.UniqueID:             id,
		}
		if b.hasTimeRemaining() {
			d.Components[id][discovery.JSONAttributesTopic] = b.Topic()
			d.Components[id][discovery.JSONAttributesTemplate] = "{{ iif(value_json.timeRemaining is defined, {'remaining': value_json.timeRemaining}, {}) | tojson }}"
		}
	}
	if b.flags.Has(batteryPower) {
		id = d.Origin.Name + "_battery_power"
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 "Battery power",
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "power",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           b.Topic(),
			discovery.ValueTemplate:        "{{ value_json.power }}",
			discovery.UnitOfMeasurement:    "W",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
	}
}

// CPU Discovery
func (c *CPU) discover(core int, d *discovery.Discovery) {
	var id, name, template string
	avail := availabilityTemplate(c.Topic())
	const bitCount = 32 << (^uint(0) >> 63)
	if c.flags.Has(cpuUsage) {
		if core == -1 {
			id = d.Origin.Name + "_cpu"
			name = "CPU usage"
			template = "{{ value_json.usage }}"
		} else {
			id = d.Origin.Name + "_cpu_core_" + strconv.Itoa(core)
			name = "Core " + strconv.Itoa(core) + " usage"
			template = fmt.Sprintf("{{ value_json[%d].usage }}", core)
		}
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 name,
			discovery.Icon:                 icon.CPU,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.StateTopic:           c.Topic(),
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.ValueTemplate:        template,
			discovery.UnitOfMeasurement:    "%",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     core == -1,
		}
	}
	if c.flags.Has(cpuTemperature) {
		if core == -1 {
			id = d.Origin.Name + "_cpu_temperature"
			name = "CPU temperature"
			template = "{{ value_json.temperature }}"
		} else {
			id = d.Origin.Name + "_cpu_core_" + strconv.Itoa(core) + "_temperature"
			name = "Core " + strconv.Itoa(core) + " temperature"
			template = fmt.Sprintf("{{ value_json.cores[%d].temperature }}", core)
		}
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 name,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "temperature",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           c.Topic(),
			discovery.ValueTemplate:        template,
			discovery.UnitOfMeasurement:    "°C",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     core == -1,
		}
	}
	if c.flags.Has(cpuFrequency) {
		if core == -1 {
			id = d.Origin.Name + "_cpu_frequency"
			name = "CPU frequency"
			template = "{{ value_json.frequency }}"
		} else {
			id = d.Origin.Name + "_cpu_core_" + strconv.Itoa(core) + "_frequency"
			name = "Core " + strconv.Itoa(core) + " frequency"
			template = fmt.Sprintf("{{ value_json.cores[%d].frequency }}", core)
		}
		d.Components[id] = discovery.Component{
			discovery.Platform:                  discovery.Sensor,
			discovery.Name:                      name,
			discovery.EntityCategory:            discovery.Diagnostic,
			discovery.DeviceClass:               "frequency",
			discovery.StateTopic:                c.Topic(),
			discovery.AvailabilityTopic:         d.AvailabilityTopic,
			discovery.AvailabilityTemplate:      avail,
			discovery.ValueTemplate:             template,
			discovery.UnitOfMeasurement:         "GHz",
			discovery.SuggestedDisplayPrecision: 3,
			discovery.UniqueID:                  id,
			discovery.EnabledByDefault:          core == -1,
		}
	}
}

func (c *CPU) Discover(d *discovery.Discovery) {
	c.discover(-1, d)
	for i := range c.cores {
		c.discover(c.cores[i].logical, d)
	}
}

// Directory Discovery
func (d *Dir) Discover(disc *discovery.Discovery) {
	id := disc.Origin.Name + "_dir_" + d.Slug()
	avail := availabilityTemplate(d.Topic())
	disc.Components[id] = discovery.Component{
		discovery.Platform:               discovery.Sensor,
		discovery.Name:                   "Dir " + d.Name,
		discovery.Icon:                   icon.Folder,
		discovery.EntityCategory:         discovery.Diagnostic,
		discovery.DeviceClass:            "data_size",
		discovery.AvailabilityTopic:      disc.AvailabilityTopic,
		discovery.AvailabilityTemplate:   avail,
		discovery.StateTopic:             d.Topic(),
		discovery.ValueTemplate:          "{{ value_json.size }}",
		discovery.UnitOfMeasurement:      d.byteSize,
		discovery.JSONAttributesTopic:    d.Topic(),
		discovery.JSONAttributesTemplate: "{{ {'path': value_json.path} | tojson }}",
		discovery.UniqueID:               id,
	}
}

// Disk Discovery
func (d *Disk) discover(dsks *Disks, disc *discovery.Discovery) {
	id := disc.Origin.Name + "_disk_" + d.Name
	name := "Disk " + d.Name
	avail := availabilityTemplate(dsks.Topic())
	disc.Components[id] = discovery.Component{
		discovery.Platform:                  discovery.Sensor,
		discovery.Name:                      name,
		discovery.Icon:                      icon.HDD,
		discovery.EntityCategory:            discovery.Diagnostic,
		discovery.AvailabilityTopic:         disc.AvailabilityTopic,
		discovery.AvailabilityTemplate:      avail,
		discovery.StateTopic:                dsks.Topic(),
		discovery.ValueTemplate:             fmt.Sprintf("{{ 100 * value_json[%[1]q].used / value_json[%[1]q].total }}", d.Name),
		discovery.UnitOfMeasurement:         "%",
		discovery.SuggestedDisplayPrecision: 1,
		discovery.JSONAttributesTopic:       dsks.Topic(),
		discovery.JSONAttributesTemplate: fmt.Sprintf(
			"{{ dict(value_json[%q]|items|rejectattr('0', 'in', ['reads', 'writes'])|list + [('size_unit', %q)]) | tojson }}",
			d.Name,
			d.size,
		),
		discovery.UniqueID: id,
	}
	if d.showIO {
		id = disc.Origin.Name + "_disk_" + d.Name + "_rx"
		disc.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 name + " rx",
			discovery.Icon:                 icon.HDD,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_size",
			discovery.AvailabilityTopic:    disc.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           dsks.Topic(),
			discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].reads }}", d.Name),
			discovery.UnitOfMeasurement:    "B",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
		id = disc.Origin.Name + "_disk_" + d.Name + "_tx"
		disc.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 name + " tx",
			discovery.Icon:                 icon.HDD,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_size",
			discovery.AvailabilityTopic:    disc.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           dsks.Topic(),
			discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].writes }}", d.Name),
			discovery.UnitOfMeasurement:    "B",
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
	}
}

func (d *Disks) Discover(disc *discovery.Discovery) {
	for _, dsk := range d.disks {
		dsk.discover(d, disc)
	}
}

// Memory Discovery
func (m *Memory) Discover(d *discovery.Discovery) {
	id := d.Origin.Name + "_memory"
	avail := availabilityTemplate(m.Topic())
	d.Components[id] = discovery.Component{
		discovery.Platform:                  discovery.Sensor,
		discovery.Name:                      "Memory usage",
		discovery.Icon:                      icon.Memory,
		discovery.EntityCategory:            discovery.Diagnostic,
		discovery.AvailabilityTopic:         d.AvailabilityTopic,
		discovery.AvailabilityTemplate:      avail,
		discovery.StateTopic:                m.Topic(),
		discovery.ValueTemplate:             "{{ 100 * value_json.used / value_json.total }}",
		discovery.UnitOfMeasurement:         "%",
		discovery.SuggestedDisplayPrecision: 1,
		discovery.JSONAttributesTopic:       m.Topic(),
		discovery.JSONAttributesTemplate: fmt.Sprintf(
			"{{ dict(value_json|items|rejectattr('0', 'match', '^swap')|list + [('size_unit', %q)]) | tojson }}",
			m.size,
		),
		discovery.UniqueID: id,
	}
	id = d.Origin.Name + "_memory_total"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Memory total",
		discovery.Icon:                 icon.Memory,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           m.Topic(),
		discovery.ValueTemplate:        "{{ value_json.total }}",
		discovery.UnitOfMeasurement:    m.size,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	id = d.Origin.Name + "_memory_used"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Memory used",
		discovery.Icon:                 icon.Memory,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           m.Topic(),
		discovery.ValueTemplate:        "{{ value_json.used }}",
		discovery.UnitOfMeasurement:    m.size,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	id = d.Origin.Name + "_memory_free"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Memory free",
		discovery.Icon:                 icon.Memory,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           m.Topic(),
		discovery.ValueTemplate:        "{{ value_json.free }}",
		discovery.UnitOfMeasurement:    m.size,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	id = d.Origin.Name + "_memory_cached"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Memory cached",
		discovery.Icon:                 icon.Memory,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           m.Topic(),
		discovery.ValueTemplate:        "{{ value_json.cached }}",
		discovery.UnitOfMeasurement:    m.size,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	if m.includeSwap {
		id = d.Origin.Name + "_memory_swap"
		d.Components[id] = discovery.Component{
			discovery.Platform:                  discovery.Sensor,
			discovery.Name:                      "Swap usage",
			discovery.Icon:                      icon.Database,
			discovery.EntityCategory:            discovery.Diagnostic,
			discovery.AvailabilityTopic:         d.AvailabilityTopic,
			discovery.AvailabilityTemplate:      avail,
			discovery.StateTopic:                m.Topic(),
			discovery.ValueTemplate:             "{{ 100 * value_json.swapUsed / value_json.swapTotal }}",
			discovery.UnitOfMeasurement:         "%",
			discovery.SuggestedDisplayPrecision: 1,
			discovery.JSONAttributesTopic:       m.Topic(),
			discovery.JSONAttributesTemplate: fmt.Sprintf(
				"{{ {'total': value_json.swapTotal, 'used': value_json.swapUsed, 'free': value_json.swapFree, 'size_unit': %q} | tojson }}",
				m.swapSize,
			),
			discovery.UniqueID: id,
		}
		id = d.Origin.Name + "_memory_swap_total"
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 "Swap total",
			discovery.Icon:                 icon.Database,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_size",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           m.Topic(),
			discovery.ValueTemplate:        "{{ value_json.swapTotal }}",
			discovery.UnitOfMeasurement:    m.swapSize,
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
		id = d.Origin.Name + "_memory_swap_used"
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 "Swap used",
			discovery.Icon:                 icon.Database,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_size",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           m.Topic(),
			discovery.ValueTemplate:        "{{ value_json.swapUsed }}",
			discovery.UnitOfMeasurement:    m.swapSize,
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
		id = d.Origin.Name + "_memory_swap_free"
		d.Components[id] = discovery.Component{
			discovery.Platform:             discovery.Sensor,
			discovery.Name:                 "Swap free",
			discovery.Icon:                 icon.Database,
			discovery.EntityCategory:       discovery.Diagnostic,
			discovery.DeviceClass:          "data_size",
			discovery.AvailabilityTopic:    d.AvailabilityTopic,
			discovery.AvailabilityTemplate: avail,
			discovery.StateTopic:           m.Topic(),
			discovery.ValueTemplate:        "{{ value_json.swapFree }}",
			discovery.UnitOfMeasurement:    m.swapSize,
			discovery.UniqueID:             id,
			discovery.EnabledByDefault:     false,
		}
	}
}

// Network Discovery
func (iface *NetInterface) discover(name string, n *Net, d *discovery.Discovery) {
	id := d.Origin.Name + "_net_" + name + "_rx"
	avail := availabilityTemplate(n.Topic())
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Network " + name + " rx rate",
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_rate",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           n.Topic(),
		discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].download_rate|default(0) }}", name),
		discovery.UnitOfMeasurement:    iface.rate,
		discovery.UniqueID:             id,
	}
	id = id[:len(id)-2] + "tx"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Network " + name + " tx rate",
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_rate",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           n.Topic(),
		discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].upload_rate|default(0) }}", name),
		discovery.UnitOfMeasurement:    iface.rate,
		discovery.UniqueID:             id,
	}
	id = d.Origin.Name + "_net_" + name + "_rx_bytes"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Network " + name + " rx bytes",
		discovery.Icon:                 icon.ServerNetwork,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           n.Topic(),
		discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].download }}", name),
		discovery.UnitOfMeasurement:    byteutil.Bytes,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
	id = d.Origin.Name + "_net_" + name + "_tx_bytes"
	d.Components[id] = discovery.Component{
		discovery.Platform:             discovery.Sensor,
		discovery.Name:                 "Network " + name + " tx bytes",
		discovery.Icon:                 icon.ServerNetwork,
		discovery.EntityCategory:       discovery.Diagnostic,
		discovery.DeviceClass:          "data_size",
		discovery.AvailabilityTopic:    d.AvailabilityTopic,
		discovery.AvailabilityTemplate: avail,
		discovery.StateTopic:           n.Topic(),
		discovery.ValueTemplate:        fmt.Sprintf("{{ value_json[%q].upload }}", name),
		discovery.UnitOfMeasurement:    byteutil.Bytes,
		discovery.UniqueID:             id,
		discovery.EnabledByDefault:     false,
	}
}

func (n *Net) Discover(d *discovery.Discovery) {
	for name, iface := range n.interfaces {
		iface.discover(name, n, d)
	}
}
