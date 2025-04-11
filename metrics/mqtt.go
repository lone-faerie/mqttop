package metrics

import (
	"fmt"
	"strconv"

	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/discovery/icon"
	"github.com/lone-faerie/mqttop/internal/byteutil"
)

func availabilityTemplate(topic string) string {
	return fmt.Sprintf(
		"{{ iif(value_json[%q]|default, 'online', 'offline') if value_json is defined else value }}",
		topic,
	)
}

// Battery Discovery

// Discover implements [discovery.Discoverer]. Adds sensors for battery state,
// battery level, battery power, and a binary sensor for battery charging.
func (b *Battery) Discover(d *discovery.Discovery) {
	id := d.Origin.Name + "_battery_state"
	avail := availabilityTemplate(b.Topic())

	var cmps []string

	if d.Nodes != nil {
		node, ok := d.Nodes[b.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 4)
		}

		cmps = node
	}

	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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

	if cmps != nil {
		d.Nodes[b.Type()] = cmps
	}
}

// CPU Discovery

func (c *CPU) discover(core int, d *discovery.Discovery) {
	var (
		id, name, template string
		avail              = availabilityTemplate(c.Topic())
		cmps               []string
	)

	if d.Nodes != nil {
		node, ok := d.Nodes[c.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 3)
		}

		cmps = node
	}

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

		if cmps != nil {
			cmps = append(cmps, id)
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

		if cmps != nil {
			cmps = append(cmps, id)
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

		if cmps != nil {
			cmps = append(cmps, id)
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

	if cmps != nil {
		d.Nodes[c.Type()] = cmps
	}
}

// Discover implements [discovery.Discoverer]. Adds sensors for cpu and core usage,
// cpu and core temperature, and cpu and core frequency.
func (c *CPU) Discover(d *discovery.Discovery) {
	c.discover(-1, d)

	for i := range c.cores {
		c.discover(c.cores[i].logical, d)
	}
}

// Directory Discovery

// Discover implements [discovery.Discoverer]. Adds sensors for directory size.
func (d *Dir) Discover(disc *discovery.Discovery) {
	id := disc.Origin.Name + "_dir_" + d.Slug()
	avail := availabilityTemplate(d.Topic())

	var cmps []string

	if disc.Nodes != nil {
		node, ok := disc.Nodes[d.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 1)
		}

		cmps = node
	}

	if cmps != nil {
		cmps = append(cmps, id)
	}

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

	if cmps != nil {
		disc.Nodes[d.Type()] = cmps
	}
}

// Disk Discovery

func (d *Disk) discover(dsks *Disks, disc *discovery.Discovery) {
	id := disc.Origin.Name + "_disk_" + d.Name
	name := "Disk " + d.Name
	avail := availabilityTemplate(dsks.Topic())

	var cmps []string

	if disc.Nodes != nil {
		node, ok := disc.Nodes[dsks.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 3)
		}

		cmps = node
	}

	if cmps != nil {
		cmps = append(cmps, id)
	}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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

	if cmps != nil {
		disc.Nodes[dsks.Type()] = cmps
	}
}

// Discover implements [discovery.Discoverer]. Adds sensors for disk usage, disk reads,
// and disk writes.
func (d *Disks) Discover(disc *discovery.Discovery) {
	for _, dsk := range d.disks {
		dsk.discover(d, disc)
	}
}

// Memory Discovery

// Discover implements [discovery.Discoverer]. Adds sensors for memory usage,
// total memory, used memory, free memory, cached memory, swap usage,
// total swap, used swap, and free swap.
func (m *Memory) Discover(d *discovery.Discovery) {
	id := d.Origin.Name + "_memory"
	avail := availabilityTemplate(m.Topic())

	var cmps []string

	if d.Nodes != nil {
		node, ok := d.Nodes[m.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 9)
		}

		cmps = node
	}

	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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
		if cmps != nil {
			cmps = append(cmps, id)
		}

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

	if cmps != nil {
		d.Nodes[m.Type()] = cmps
	}
}

// Network Discovery

func (iface *NetInterface) discover(name string, n *Net, d *discovery.Discovery) {
	id := d.Origin.Name + "_net_" + name + "_rx"
	avail := availabilityTemplate(n.Topic())

	var cmps []string

	if d.Nodes != nil {
		node, ok := d.Nodes[n.Type()]
		if !ok || node == nil {
			node = make([]string, 0, 4)
		}

		cmps = node
	}

	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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
	if cmps != nil {
		cmps = append(cmps, id)
	}

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

	if cmps != nil {
		d.Nodes[n.Type()] = cmps
	}
}

// Discover implements [discovery.Discoverer]. Adds sensors for interface rx rate,
// tx rate, rx bytes, and tx bytes.
func (n *Net) Discover(d *discovery.Discovery) {
	for name, iface := range n.interfaces {
		iface.discover(name, n, d)
	}
}
