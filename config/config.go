// Package config provides the structures used for configuration.
package config

import (
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lone-faerie/mqttop/config/secrets"
	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/log"
)

// Config contains the configuration for the MQTT client and metrics.
// Config should be created with a call to [Default], [Read], or [Load] as
// some options require further configuration than simply setting.
type Config struct {
	Interval    time.Duration   `yaml:"interval"`
	TopicPrefix string          `yaml:"topic_prefix"`
	MQTT        MQTTConfig      `yaml:"mqtt,omitempty"`
	Discovery   DiscoveryConfig `yaml:"discovery,omitempty"`
	Log         LogConfig       `yaml:"log,omitempty"`
	CPU         CPUConfig       `yaml:"cpu,omitempty"`
	Memory      MemoryConfig    `yaml:"memory,omitempty"`
	Disks       DisksConfig     `yaml:"disks,omitempty"`
	Net         NetConfig       `yaml:"net,omitempty"`
	Battery     BatteryConfig   `yaml:"battery,omitempty"`
	Dirs        []DirConfig     `yaml:"dirs,omitempty"`
	GPU         GPUConfig       `yaml:"gpu,omitempty"`

	FormatSize func(v int, bits bool) string `yaml:"-"`
}

var defaultCfg = &Config{
	Interval:    2 * time.Second,
	TopicPrefix: "mqttop",
	MQTT:        defaultMQTT,
	Discovery:   defaultDiscovery,
	CPU:         defaultCPU,
	Memory:      defaultMemory,
	Disks:       defaultDisks,
	Net:         defaultNet,
	Battery:     defaultBattery,
	GPU:         defaultGPU,
}

// Default returns the default Config when no config file is provided.
func Default() *Config {
	cfg := defaultCfg
	cfg.load()
	return cfg
}

// Read returns the Config parsed from the yaml encoded config from r.
func Read(r io.Reader) (cfg *Config, err error) {
	cfg = defaultCfg
	if err = yaml.NewDecoder(r).Decode(cfg); err != nil {
		return
	}
	err = cfg.load()
	return
}

// Load returns the Config parsed from the given yaml files. If the first file does
// not exist, the default config is returned. If any of the given paths are
// directories, all the files in the directory are read.
func Load(file ...string) (cfg *Config, err error) {
	log.Info("Loading config", "path", file)
	if _, err = os.Stat(file[0]); err != nil {
		return defaultCfg, nil
	}
	r := byteutil.NewMultiFileReader(file...)
	defer r.Close()
	return Read(r)
}

func (cfg *Config) loadValue(v reflect.Value) error {
	iface := v.Addr().Interface()
	if l, ok := iface.(loader); ok {
		return l.load(cfg)
	}
	return nil
}

func (cfg *Config) load() (err error) {
	log.Debug("Topic Prefix", "prefix", cfg.TopicPrefix)
	if cfg.TopicPrefix != "mqttop" {
		log.Debug("Replacing topic prefix", "old", "mqttop", "new", cfg.TopicPrefix)
		if s, ok := strings.CutPrefix(cfg.MQTT.BirthWillTopic, "mqttop/"); ok {
			log.Debug("Replacing prefix of birth_lwt_topic", "prefix", cfg.TopicPrefix)
			cfg.MQTT.BirthWillTopic = cfg.TopicPrefix + "/" + s
		}
		if s, ok := strings.CutPrefix(cfg.Discovery.Availability, "mqttop/"); ok {
			cfg.Discovery.Availability = cfg.TopicPrefix + "/" + s
		}
	}
	var (
		v = reflect.ValueOf(cfg).Elem()
		n = v.NumField()
	)
	//	expandValue(v)
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
		if cfg.TopicPrefix != "mqttop" && slices.Contains(topicFields, field) {
			if topic, ok := strings.CutPrefix(s, "mqttop/"); ok {
				s = cfg.TopicPrefix + "/" + topic
			}
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

func expandValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		s := Expand(v.String())
		v.SetString(s)
	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			expandValue(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		for i := 0; i < n; i++ {
			expandValue(v.Index(i))
		}

	case reflect.Pointer:
		expandValue(v.Elem())
	}
}

func replacePrefix(v reflect.Value, prefix string) {
	switch v.Kind() {
	case reflect.Struct:
		f := v.FieldByName("Topic")
		if f.IsValid() && f.Kind() == reflect.String {
			if s, ok := strings.CutPrefix(f.String(), "mqttop/"); ok {
				f.SetString(prefix + "/" + s)
			}
			return
		}
		n := v.NumField()
		for i := 0; i < n; i++ {
			replacePrefix(v.Field(i), prefix)
		}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		for i := 0; i < n; i++ {
			replacePrefix(v.Index(i), prefix)
		}
	case reflect.Pointer:
		replacePrefix(v.Elem(), prefix)
	}
}

// Expand replaces ${var} or $var in s according to the values of
// the current environment variables, and replaces !secret var according
// to the file at /run/secret/<var>.
func Expand(s string) string {
	if secret, ok := secrets.CutPrefix(s); ok {
		return secrets.MustRead(secret, "")
	}
	return os.ExpandEnv(s)
}

// Expand calls [Expand] on every string field of cfg.
func (cfg *Config) Expand() {
	v := reflect.ValueOf(cfg).Elem()
	expandValue(v)
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
			return
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
