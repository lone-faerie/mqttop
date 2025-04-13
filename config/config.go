// Package config provides the structures used for configuration.
//
// Configuration can be loaded from multiple YAML files, including from directories.
// If no config file is specified, the default path(s) will be determined by the first
// defined value of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, or $HOME/.config/mqttop.yaml.
// In the case of $MQTTOP_CONFIG_PATH, the value may be a comma-separated list of paths. If none of
// these files exist, the default configuration will be used, which looks for the following
// environment variables:
//
//   - broker:   $MQTTOP_BROKER_ADDRESS
//   - username: $MQTTOP_BROKER_USERNAME
//   - password: $MQTTOP_BROKER_PASSWORD
package config

import (
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lone-faerie/mqttop/config/secrets"
	"github.com/lone-faerie/mqttop/internal/file"
	"github.com/lone-faerie/mqttop/log"
)

// Config contains the configuration for the MQTT client and metrics.
// Config should be created with a call to [Default], [Read], or [Load] as
// some options require further configuration than simply setting.
type Config struct {
	// Interval is the default update interval for all enabled metrics.
	// Any metric with an update interval of 0 will use Interval instead.
	Interval time.Duration `yaml:"interval"`
	// BaseTopic is a value that may be used multiple times in configuration.
	// If the options "birth_lwt_topic" for MQTT configuration, "availability"
	// for discovery configuration, or "topic" for any metric configuration
	// have the prefix or suffix of "~" then that "~" will be replaced with
	// BaseTopic. The default value is "mqttop".
	//
	// For example if BaseTopic is "foo" then
	// "~/bridge/status" becomes "foo/bridge/status"
	BaseTopic string `yaml:"base_topic"`

	MQTT      MQTTConfig      `yaml:"mqtt,omitempty"`
	Discovery DiscoveryConfig `yaml:"discovery,omitempty"`
	Log       LogConfig       `yaml:"log,omitempty"`
	CPU       CPUConfig       `yaml:"cpu,omitempty"`
	Memory    MemoryConfig    `yaml:"memory,omitempty"`
	Disks     DisksConfig     `yaml:"disks,omitempty"`
	Net       NetConfig       `yaml:"net,omitempty"`
	Battery   BatteryConfig   `yaml:"battery,omitempty"`
	Dirs      []DirConfig     `yaml:"dirs,omitempty"`
	GPU       GPUConfig       `yaml:"gpu,omitempty"`
}

func defaultCfg() *Config {
	return &Config{
		Interval:  2 * time.Second,
		BaseTopic: "mqttop",
		MQTT:      DefaultMQTT,
		Discovery: DefaultDiscovery,
		CPU:       DefaultCPU,
		Memory:    DefaultMemory,
		Disks:     DefaultDisks,
		Net:       DefaultNet,
		Battery:   DefaultBattery,
		GPU:       DefaultGPU,
	}
}

// Default returns the default configuration,
//
//	Config{
//		Interval:    2 * time.Second,
//		TopicPrefix: "mqttop",
//		MQTT:        DefaultMQTT,
//		Discovery:   DefaultDiscovery,
//		CPU:         DefaultCPU,
//		Memory:      DefaultMemory,
//		Disks:       DefaultDisks,
//		Net:         DefaultNet,
//		Battery:     DefaultBattery,
//		GPU:         DefaultGPU,
//	}
func Default() *Config {
	cfg := defaultCfg()
	cfg.init()

	return cfg
}

// Read returns the Config parsed from the yaml encoded config from r.
func Read(r io.Reader) (cfg *Config, err error) {
	cfg = defaultCfg()
	if err = yaml.NewDecoder(r).Decode(cfg); err != nil {
		return
	}

	err = cfg.init()

	return
}

func hasNonYAML(filenames []string) bool {
	for _, name := range filenames {
		switch filepath.Ext(name) {
		case "", ".yml", ".yaml":
		default:
			return true
		}
	}

	return false
}

// Load returns the Config parsed from the given yaml files. If the first file does
// not exist, the default config is returned. If any of the given paths are
// directories, all the files in the directory are read. If none of the given filenames
// have an extension, they are assumed to be directories and only files with the
// extensions ".yml" or ".yaml" will be read.
func Load(filename ...string) (cfg *Config, err error) {
	log.Info("Loading config", "path", filename)

	if len(filename) == 0 {
		return Default(), nil
	}

	if _, err = os.Stat(filename[0]); err != nil {
		return Default(), nil
	}

	r := file.NewMultiReader(filename...)
	if !hasNonYAML(filename) {
		r.WithExtension(".yml", ".yaml")
	}

	defer r.Close()
	cfg, err = Read(r)

	if err == io.EOF {
		err = nil
	}

	return
}

// ReplaceBase returns topic with the prefix and/or suffix "~" replaced with base.
// ReplaceBase returns topic, and if topic is not prefixed or suffixed with "~" or
// if either topic or base are an empty string.
func ReplaceBase(base, topic string) string {
	if topic == "" || base == "" {
		return topic
	}
	if topic[0] == '~' {
		topic = base + topic[1:]
	}
	if topic[len(topic)-1] == '~' {
		topic = topic[:len(topic)-1] + base
	}
	return topic
}

func (cfg *Config) init() (err error) {
	if cfg.BaseTopic != "" {
		log.Debug("Replacing base topic", "old", "~", "new", cfg.BaseTopic)

		cfg.MQTT.BirthWillTopic = ReplaceBase(cfg.BaseTopic, cfg.MQTT.BirthWillTopic)
		cfg.Discovery.Availability = ReplaceBase(cfg.BaseTopic, cfg.Discovery.Availability)
	}

	var (
		v = reflect.ValueOf(cfg).Elem()
		n = v.NumField()
	)
	for i := 0; i < n; i++ {
		cfg.forValue(v.Field(i), "")
	}

	return
}

var topicFields = []string{
	"BirthWillTopic", "Availability", "Topic",
}

func (cfg *Config) forValue(v reflect.Value, field string) {
	switch v.Kind() {
	case reflect.String:
		s := Expand(v.String())
		if s != "" && cfg.BaseTopic != "" && slices.Contains(topicFields, field) {
			s = ReplaceBase(cfg.BaseTopic, s)
		}

		v.SetString(s)
	case reflect.Struct:
		iface := v.Addr().Interface()
		if l, ok := iface.(loader); ok {
			l.load(cfg)
		}

		t := v.Type()
		n := v.NumField()

		for i := 0; i < n; i++ {
			f := t.Field(i)
			cfg.forValue(v.FieldByIndex(f.Index), f.Name)
		}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		for i := 0; i < n; i++ {
			cfg.forValue(v.Index(i), "")
		}
	case reflect.Pointer:
		cfg.forValue(v.Elem(), "")
	}
}

// Expand replaces "!secret var" according to the file at /run/secret/<var>
// and replaces ${var} or $var in s according to the values of the current
// environment variables.
func Expand(s string) string {
	if secret, ok := secrets.CutPrefix(s); ok {
		return secrets.MustRead(secret, "")
	}

	return os.ExpandEnv(s)
}

// Write writes the yaml encoding of cfg to w.
func (cfg *Config) Write(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer enc.Close()

	enc.SetIndent(2)

	return enc.Encode(cfg)
}

func setInterval(v reflect.Value, d time.Duration) {
	switch v.Kind() {
	case reflect.Pointer:
		setInterval(v.Elem(), d)
	case reflect.Slice:
		n := v.Len()
		for i := 0; i < n; i++ {
			setInterval(v.Index(i), d)
		}
	case reflect.Struct:
		if f := v.FieldByName("Interval"); f.IsValid() && f.Kind() == reflect.Int64 {
			f.SetInt(int64(d))
		}

		n := v.NumField()
		for i := 0; i < n; i++ {
			setInterval(v.Field(i), d)
		}
	}
}

// SetInterval sets the update interval for every metric config.
func (cfg *Config) SetInterval(d time.Duration) {
	setInterval(reflect.ValueOf(cfg).Elem(), d)
}

// SetMetrics enables each of the given metrics and disables all others.
// If only the value "all" is given, all metrics will be enabled.
func (cfg *Config) SetMetrics(name ...string) {
	enableAll := len(name) == 1 && name[0] == "all"
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()
	n := t.NumField()

	for i := 0; i < n; i++ {
		f := t.Field(i)
		if f.Type.Kind() != reflect.Struct {
			continue
		}

		if _, ok := f.Type.FieldByName("MetricConfig"); !ok {
			continue
		}

		tag, _, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		enabled := enableAll || slices.Contains(name, tag)
		v.FieldByIndex(f.Index).FieldByName("MetricConfig").FieldByName("Enabled").SetBool(enabled)
	}
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
