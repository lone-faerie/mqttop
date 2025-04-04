package config

import (
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

type loader interface {
	load(*Config) error
}

// MetricConfig is the base configuration of any metric.
type MetricConfig struct {
	Enabled bool `yaml:"enabled"`
	// Interval is the update interval of the metric. If 0 then
	// the Interval of the parent [Config] is used.
	Interval time.Duration `yaml:"interval,omitempty"`
	// Topic is the topic updates for the metric are published to.
	// The default value is "mqttop/metric/<metric_type>"
	Topic string `yaml:"topic,omitempty"`
}

// CPUConfig is the configuration for the CPU metrics.
type CPUConfig struct {
	MetricConfig `yaml:",inline"`

	// Name is a custom name used for the CPU. If blank (default) then
	// the name is the model name in /proc/cpuinfo.
	Name string `yaml:"name,omitempty"`
	// NameTemplate is a template used for rendering a custom name for the CPU.
	// If not blank then the rendered value will override Name.
	// See https://pkg.go.dev/text/template
	NameTemplate string `yaml:"name_template,omitempty"`
	// SelectionMode is the mode used to select the overall CPU temperature and frequency.
	// and frequency. The acceptable values are:
	//	- "auto"    (package temperature, frequency of first core)
	//	- "first"   (values of first core)
	//	- "average" (average of all cores)
	//	- "max"     (maximum of all cores)
	//	- "min"     (minimum of all cores)
	//	- "random"  (value of random core)
	SelectionMode string `yaml:"selection_mode,omitempty"`

	nameTemplate *template.Template
}

// MemoryConfig is the configuration for the memory metrics.
type MemoryConfig struct {
	MetricConfig `yaml:",inline"`

	// SizeUnit is the unit to use when reporting the size. If blank
	// then the unit will automatically be determined. The acceptable
	// values are:
	//	- "Bytes", "bytes", or "B"
	//	- "KiB"
	//	- "MiB"
	//	- "GiB"
	//	- "TiB"
	//	- "PiB"
	SizeUnit string `yaml:"size_unit,omitempty"`
	// IncludeSwap indicates if the swap memory should be included
	// in the metrics.
	IncludeSwap bool `yaml:"include_swap,omitempty"`
}

// DiskConfig is the configuration for an individual disk's metrics.
type DiskConfig struct {
	MetricConfig `yaml:",inline"`

	// Exclude indicates if the disk should be excluded.
	Exclude bool `yaml:"exclude,omitempty"`
	// Name is a custom name used for the disk. If blank (default)
	// then the name will be the base path of mount point.
	Name string `yaml:"name,omitempty"`
	// NameTemplate is a template used for rendering a custom name for the disk.
	// If not blank then the rendered value will override Name.
	// See https://pkg.go.dev/text/template
	NameTemplate string `yaml:"name_template,omitempty"`
	// MountPoint is the mount point (path) of the disk.
	MountPoint string `yaml:"mount,omitempty"`
	// SizeUnit is the unit to use when reporting the size. If blank
	// then the unit will automatically be determined. The acceptable
	// values are:
	//	- "Bytes", "bytes", or "B"
	//	- "KiB"
	//	- "MiB"
	//	- "GiB"
	//	- "TiB"
	//	- "PiB"
	SizeUnit string `yaml:"size_unit,omitempty"`
	// ShowIO indicates if IO operations (reads/writes) should be included in
	// the metrics.
	ShowIO bool `yaml:"show_io,omitempty"`

	nameTemplate *template.Template
}

// DisksConfig is the configuration for the disks metrics.
type DisksConfig struct {
	MetricConfig `yaml:",inline"`

	// UseFSTab indicates if /etc/fstab should be used to determine disks
	// on the system.
	UseFSTab bool `yaml:"use_fstab"`
	// Rescan is the interval at which to rescan for disks. If the value can
	// be parsed as a boolean, then false (default) will not perform rescans
	// and true will set the rescan interval to the update interval. Otherwise
	// the value is parsed as a [time.Duration].
	Rescan string `yaml:"rescan,omitempty"`
	// ShowIO indicates if IO operations (reads/writes) should be included in
	// the metrics.
	ShowIO bool `yaml:"show_io"`
	// Disk is a list of configurations for each individual disk.
	Disk []DiskConfig `yaml:"disk,omitempty"`

	// RescanInterval is the interval parsed from Rescan
	RescanInterval time.Duration `yaml:"-"`
	diskMap        map[string]*DiskConfig
}

// NetIfaceConfig is the configuration for an individual network interface.
type NetIfaceConfig struct {
	// Name is a custom name used for the interface. If blank (default)
	// then the name will be the name reported by the system.
	Name string `yaml:"name,omitempty"`
	// NameTemplate is a template used for rendering a custom name for the
	// interface. If not blank then the rendered value will override Name.
	// See https://pkg.go.dev/text/template
	NameTemplate string `yaml:"name_template,omitempty"`
	// Interface is the name of the interface as reported by the system.
	Interface string `yaml:"interface,omitempty"`
	// RateUnit is the unit to use when reporting the data rate. The default
	// value is the RateUnit of the parent [NetConfig]. The acceptable
	// values are:
	//	- "Bytes/s", "bytes/s", "B/s", or "Bps"
	//	- "KiB/s" or "KiBps"
	//	- "MiB/s" or "MiBps"
	//	- "GiB/s" or "GiBps"
	//	- "TiB/s" or "TiBps"
	//	- "PiB/s" or "PiBps"
	RateUnit string `yaml:"rate_unit,omitempty"`

	nameTemplate *template.Template
}

// NetConfig is the configuration for the network metrics.
type NetConfig struct {
	MetricConfig `yaml:",inline"`

	// OnlyPhysical indicates if only physical interfaces should be included.
	OnlyPhysical bool `yaml:"only_physical"`
	// OnlyRunning indicates if only running interfaces should be included.
	OnlyRunning bool `yaml:"only_running"`
	// IncludeBridge indicates if interfaces of type bridge should be included.
	IncludeBridge bool `yaml:"include_bridge"`
	// Rescan is the interval at which to rescan for interfaced. If the value can
	// be parsed as a boolean, then false (default) will not perform rescans
	// and true will set the rescan interval to the update interval. Otherwise
	// the value is parsed as a [time.Duration].
	Rescan string `yaml:"rescan,omitempty"`
	// RateUnit is the unit to use when reporting the data rate. The default
	// value is "MiB/s". The acceptable values are:
	//	- "Bytes/s", "bytes/s", "B/s", or "Bps"
	//	- "KiB/s" or "KiBps"
	//	- "MiB/s" or "MiBps"
	//	- "GiB/s" or "GiBps"
	//	- "TiB/s" or "TiBps"
	//	- "PiB/s" or "PiBps"
	RateUnit string `yaml:"rate_unit,omitempty"`
	// Include is a list of interfaces to include. If defined then only these interfaces
	// will be included. If parsed from a list of strings then the Interface field of each
	// NetIfaceConfig will be the value from the list.
	Include []NetIfaceConfig `yaml:"include,omitempty"`
	// Exclude is a list of interfaces to exclude. If defined then these interfaces will
	// not be included.
	Exclude []string `yaml:"exclude,omitempty"`

	// RescanInterval is the interval parsed from Rescan
	RescanInterval time.Duration `yaml:"-"`
}

// BatteryConfig is the configuration for the battery metrics.
type BatteryConfig struct {
	MetricConfig `yaml:",inline"`

	// TimeFormat is the format used when rendering the amount of time
	// remianing on the battery.
	// See https://pkg.go.dev/time#pkg-constants
	TimeFormat string `yaml:"time_format,omitempty"`
}

// DirConfig is the configuration for directory metrics.
type DirConfig struct {
	MetricConfig `yaml:",inline"`

	// Name is a custom name used for the directory. If blank (default)
	// then the name will be the path of the directory.
	Name string `yaml:"name,omitempty"`
	// NameTemplate is a template used for rendering a custom name for the
	// directory. If not blank then the rendered value will override Name.
	// See https://pkg.go.dev/text/template
	NameTemplate string `yaml:"name_template,omitempty"`
	// Path is the path to the directory.
	Path string `yaml:"path,omitempty"`
	// SizeUnit is the unit to use when reporting the size. If blank
	// then the unit will automatically be determined. The acceptable
	// values are:
	//	- "Bytes", "bytes", or "B"
	//	- "KiB"
	//	- "MiB"
	//	- "GiB"
	//	- "TiB"
	//	- "PiB"
	SizeUnit string `yaml:"size_unit,omitempty"`
	// Watch indicates if the directory should be watched for updates instead of polled.
	// If true then updates will be published no more than the update interval.
	Watch bool `yaml:"watch"`
	// Depth is the maximum depth to watch for updates in the directory.
	Depth int `yaml:"depth,omitempty"`

	nameTemplate *template.Template
}

// GPUConfig is the configuration for the GPU metrics.
type GPUConfig struct {
	MetricConfig `yaml:",inline"`

	// Name is a custom name used for the directory. If blank (default)
	// then the name will be the name reported by the GPU.
	Name string `yaml:"name,omitempty"`

	// NameTemplate is a template used for rendering a custom name for the
	// GPU. If not blank then the rendered value will override Name.
	// See https://pkg.go.dev/text/template
	NameTemplate string `yaml:"name_template,omitempty"`
	// Platform is the platform of the GPU to use. The acceptable values are:
	//	- "auto"
	//	- "nvidia"
	Platform string `yaml:"platform,omitempty"`
	// Index is the index of the GPU to use. The default value is 0.
	Index int `yaml:"index,omitempty"`
	// SizeUnit is the unit to use when reporting the size of memory.
	// If blank then the unit will automatically be determined. The
	// acceptable values are:
	//	- "Bytes", "bytes", or "B"
	//	- "KiB"
	//	- "MiB"
	//	- "GiB"
	//	- "TiB"
	//	- "PiB"
	SizeUnit string `yaml:"size_unit,omitempty"`
	// IncludeProcs indicates if the usage of individual processes should
	// be included in the metrics.
	// TODO(lone-faerie): not yet implemented
	IncludeProcs bool `yaml:"include_proc"`

	nameTemplate *template.Template
}

var defaultCPU = CPUConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/cpu",
	},
}

var defaultMemory = MemoryConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/memory",
	},
	IncludeSwap: true,
}

var defaultDisks = DisksConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/disk",
	},
	UseFSTab: true,
	ShowIO:   true,
}

var defaultNet = NetConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/net",
	},
	// OnlyPhysical: true,
}

var defaultBattery = BatteryConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/battery",
	},
}

var defaultGPU = GPUConfig{
	MetricConfig: MetricConfig{
		Enabled: true,
		Topic:   "mqttop/metric/gpu",
	},
}

func (cfg *CPUConfig) load(_ *Config) error {
	if !cfg.Enabled || cfg.NameTemplate == "" {
		return nil
	}
	t, err := loadTemplate("cpu_name", cfg.NameTemplate)
	if err != nil {
		return err
	}
	cfg.nameTemplate = t
	return nil
}

// FormatName returns the name rendered from the [CPUConfig].NameTemplate, if defined.
// If the template is not defined, FormatName returns name.
func (cfg *CPUConfig) FormatName(name string) string {
	if cfg.nameTemplate == nil {
		return name
	}
	var b strings.Builder
	if err := cfg.nameTemplate.Execute(&b, name); err != nil {
		return name
	}
	return b.String()
}

// UnmarshalYAML implements [yaml.Unmarshaler]. If node is a mapping then cfg is
// unmarshaled normally. Otherwise cfg is unmarshalled as a string, and cfg.MountPoint
// is set to the value of node.
func (cfg *DiskConfig) UnmarshalYAML(node *yaml.Node) error {
	type Wrapped DiskConfig
	if node.Kind&yaml.MappingNode != 0 {
		return node.Decode((*Wrapped)(cfg))
	}
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	cfg.MountPoint = s
	return nil
}

func (cfg *DisksConfig) load(c *Config) (err error) {
	cfg.diskMap = make(map[string]*DiskConfig, len(cfg.Disk))
	for i := range cfg.Disk {
		if cfg.Disk[i].NameTemplate != "" {
			t, err := loadTemplate("disk_"+cfg.Disk[i].MountPoint, cfg.Disk[i].NameTemplate)
			if err != nil {
				return err
			}
			cfg.Disk[i].nameTemplate = t
		}
		cfg.diskMap[cfg.Disk[i].MountPoint] = &cfg.Disk[i]
	}
	switch cfg.Rescan {
	case "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
		if cfg.Interval > 0 {
			cfg.RescanInterval = cfg.Interval
		} else {
			cfg.RescanInterval = c.Interval
		}
	case "false", "False", "FALSE", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
	case "":
	default:
		cfg.RescanInterval, err = time.ParseDuration(cfg.Rescan)
	}
	return
}

// Excluded returns if the configuration for mnt is set to be excluded.
func (cfg *DisksConfig) Excluded(mnt string) bool {
	dcfg, ok := cfg.diskMap[mnt]
	return ok && dcfg.Exclude
}

// ConfigFor returns the configuration for mnt.
func (cfg *DisksConfig) ConfigFor(mnt string) *DiskConfig {
	return cfg.diskMap[mnt]
}

// UnmarshalYAML implements [yaml.Unmarshaler]. If node is a mapping then cfg is
// unmarshaled normally. Otherwise cfg is unmarshalled as a string, and cfg.Interface
// is set to the value of node.
func (cfg *NetIfaceConfig) UnmarshalYAML(node *yaml.Node) error {
	type Wrapped NetIfaceConfig
	if node.Kind&yaml.MappingNode != 0 {
		return node.Decode((*Wrapped)(cfg))
	}
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	cfg.Interface = s
	return nil
}

// FormatName returns cfg.Name, if defined, or the name rendered from the cfg.NameTemplate,
// if defined. If cfg.Name and cfg.NameTemplate are not defined, FormatName returns name.
func (cfg *NetIfaceConfig) FormatName(name string) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.nameTemplate == nil {
		return name
	}
	var b strings.Builder
	if err := cfg.nameTemplate.Execute(&b, name); err != nil {
		return name
	}
	return b.String()
}

func (cfg *NetConfig) load(c *Config) (err error) {
	switch cfg.Rescan {
	case "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
		if cfg.Interval > 0 {
			cfg.RescanInterval = cfg.Interval
		} else {
			cfg.RescanInterval = c.Interval
		}
	case "false", "False", "FALSE", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
	case "":
	default:
		cfg.RescanInterval, err = time.ParseDuration(cfg.Rescan)
	}
	for i := range cfg.Include {
		if cfg.Include[i].NameTemplate == "" {
			continue
		}
		t, err := loadTemplate("net_"+cfg.Include[i].Interface, cfg.Include[i].NameTemplate)
		if err != nil {
			return err
		}
		cfg.Include[i].nameTemplate = t
	}
	return
}

func (cfg *DirConfig) load(_ *Config) (err error) {
	if cfg.NameTemplate == "" {
		return
	}
	t, err := loadTemplate("dir_"+cfg.Path, cfg.NameTemplate)
	if err != nil {
		return
	}
	cfg.nameTemplate = t
	return
}

// FormatName returns cfg.Name, if defined, or the name rendered from the cfg.NameTemplate,
// if defined. If cfg.Name and cfg.NameTemplate are not defined, FormatName returns name.
func (cfg *DirConfig) FormatName(name string) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.nameTemplate == nil {
		return name
	}
	var b strings.Builder
	if err := cfg.nameTemplate.Execute(&b, name); err != nil {
		return name
	}
	return b.String()
}

// UnmarshalYAML implements [yaml.Unmarshaler]. If node is a mapping then cfg is
// unmarshaled normally. Otherwise cfg is unmarshalled as a string, and cfg.Path
// is set to the value of node.
func (cfg *DirConfig) UnmarshalYAML(node *yaml.Node) error {
	type Wrapped DirConfig
	cfg.Depth = -1
	if node.Kind&yaml.MappingNode != 0 {
		return node.Decode((*Wrapped)(cfg))
	}
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	cfg.Path = s
	return nil
}

func (cfg *GPUConfig) load(_ *Config) error {
	if cfg.NameTemplate == "" {
		return nil
	}
	t, err := loadTemplate("gpu_name", cfg.NameTemplate)
	if err != nil {
		return err
	}
	cfg.nameTemplate = t
	return nil
}

// FormatName returns cfg.Name, if defined, or the name rendered from the cfg.NameTemplate,
// if defined. If cfg.Name and cfg.NameTemplate are not defined, FormatName returns name.
func (cfg *GPUConfig) FormatName(name string) string {
	if cfg.Name != "" {
		return cfg.Name
	}
	if cfg.nameTemplate == nil {
		return name
	}
	var b strings.Builder
	if err := cfg.nameTemplate.Execute(&b, name); err != nil {
		return name
	}
	return b.String()
}

// IsZero returns true if cfg is the default value.
func (cfg CPUConfig) IsZero() bool {
	return cfg == defaultCPU
}

// IsZero returns true if cfg is the default value.
func (cfg MemoryConfig) IsZero() bool {
	return cfg == defaultMemory
}

// IsZero returns true if cfg is the default value.
func (cfg DisksConfig) IsZero() bool {
	return cfg.MetricConfig == defaultDisks.MetricConfig &&
		cfg.UseFSTab == defaultDisks.UseFSTab &&
		cfg.Rescan == defaultDisks.Rescan &&
		cfg.ShowIO == defaultDisks.ShowIO &&
		len(cfg.Disk) == 0
}

// IsZero returns true if cfg is the default value.
func (cfg NetConfig) IsZero() bool {
	return cfg.MetricConfig == defaultNet.MetricConfig &&
		cfg.OnlyPhysical == defaultNet.OnlyPhysical &&
		cfg.OnlyRunning == defaultNet.OnlyRunning &&
		cfg.IncludeBridge == defaultNet.IncludeBridge &&
		cfg.Rescan == defaultNet.Rescan &&
		cfg.RateUnit == defaultNet.RateUnit &&
		len(cfg.Include) == 0 &&
		len(cfg.Exclude) == 0
}

// IsZero returns true if cfg is the default value.
func (cfg BatteryConfig) IsZero() bool {
	return cfg == defaultBattery
}

// IsZero returns true if cfg is the default value.
func (cfg GPUConfig) IsZero() bool {
	return cfg == defaultGPU
}
