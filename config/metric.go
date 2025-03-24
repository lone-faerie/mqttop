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

type MetricConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval,omitempty"`
	Topic    string        `yaml:"topic,omitempty"`
}

type CPUConfig struct {
	MetricConfig `yaml:",inline"`

	Name          string `yaml:"name,omitempty"`
	NameTemplate  string `yaml:"name_template,omitempty"`
	SelectionMode string `yaml:"selection_mode,omitempty"`

	nameTemplate *template.Template
}

type MemoryConfig struct {
	MetricConfig `yaml:",inline"`

	SizeUnit    string `yaml:"size_unit,omitempty"`
	IncludeSwap bool   `yaml:"include_swap,omitempty"`
}

type DiskConfig struct {
	MetricConfig `yaml:",inline"`

	Exclude      bool   `yaml:"exclude,omitempty"`
	Name         string `yaml:"name,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
	MountPoint   string `yaml:"mount,omitempty"`
	SizeUnit     string `yaml:"size_unit,omitempty"`
	ShowIO       bool   `yaml:"show_io,omitempty"`

	nameTemplate *template.Template
}

type DisksConfig struct {
	MetricConfig `yaml:",inline"`

	UseFSTab bool         `yaml:"use_fstab"`
	Rescan   string       `yaml:"rescan,omitempty"`
	ShowIO   bool         `yaml:"show_io"`
	Disk     []DiskConfig `yaml:"disk,omitempty"`

	RescanInterval time.Duration `yaml:"-"`
	diskMap        map[string]*DiskConfig
}

type NetIfaceConfig struct {
	Name         string `yaml:"name,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
	Interface    string `yaml:"interface,omitempty"`
	RateUnit     string `yaml:"rate_unit,omitempty"`

	nameTemplate *template.Template
}

type NetConfig struct {
	MetricConfig `yaml:",inline"`

	OnlyPhysical  bool             `yaml:"only_physical"`
	OnlyRunning   bool             `yaml:"only_running"`
	IncludeBridge bool             `yaml:"include_bridge"`
	Rescan        string           `yaml:"rescan,omitempty"`
	RateUnit      string           `yaml:"rate_unit,omitempty"`
	Include       []NetIfaceConfig `yaml:"include,omitempty"`
	Exclude       []string         `yaml:"exclude,omitempty"`

	RescanInterval time.Duration `yaml:"-"`
}

type BatteryConfig struct {
	MetricConfig `yaml:",inline"`

	TimeFormat string `yaml:"time_format,omitempty"`
}

type DirConfig struct {
	MetricConfig `yaml:",inline"`

	Name         string `yaml:"name,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
	Path         string `yaml:"path,omitempty"`
	SizeUnit     string `yaml:"size_unit,omitempty"`
	Watch        bool   `yaml:"watch"`
	Depth        int    `yaml:"depth,omitempty"`

	nameTemplate *template.Template
}

type GPUConfig struct {
	MetricConfig `yaml:",inline"`

	Name         string `yaml:"name,omitempty"`
	NameTemplate string `yaml:"name_template,omitempty"`
	Platform     string `yaml:"platform,omitempty"`
	Index        int    `yaml:"index,omitempty"`
	SizeUnit     string `yaml:"size_unit,omitempty"`
	IncludeProcs bool   `yaml:"include_proc"`

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

func templateFuncs() map[string]any {
	return map[string]any{
		"cut": func(s, sep string) string {
			a, b, _ := strings.Cut(s, sep)
			return a + b
		},
		"cutprefix": strings.TrimPrefix,
		"cutsuffix": strings.TrimSuffix,
		"replace":   strings.ReplaceAll,
		"tolower":   strings.ToLower,
		"totitle":   strings.ToTitle,
		"toupper":   strings.ToUpper,
		"trim":      strings.TrimSpace,
	}
}

func loadTemplate(name, text string) (*template.Template, error) {
	t := template.New(name)
	t.Funcs(templateFuncs())
	return t.Parse(text)
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

func (cfg *DisksConfig) Excluded(mnt string) bool {
	dcfg, ok := cfg.diskMap[mnt]
	return ok && dcfg.Exclude
}

func (cfg *DisksConfig) ConfigFor(mnt string) *DiskConfig {
	return cfg.diskMap[mnt]
}

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

func (cfg CPUConfig) IsZero() bool {
	return cfg == defaultCPU
}

func (cfg MemoryConfig) IsZero() bool {
	return cfg == defaultMemory
}

func (cfg DisksConfig) IsZero() bool {
	return cfg.MetricConfig == defaultDisks.MetricConfig &&
		cfg.UseFSTab == defaultDisks.UseFSTab &&
		cfg.Rescan == defaultDisks.Rescan &&
		cfg.ShowIO == defaultDisks.ShowIO &&
		len(cfg.Disk) == 0
}

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

func (cfg BatteryConfig) IsZero() bool {
	return cfg == defaultBattery
}

func (cfg GPUConfig) IsZero() bool {
	return cfg == defaultGPU
}
