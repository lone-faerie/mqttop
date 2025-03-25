package config

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/lone-faerie/mqttop/internal/byteutil"
	"github.com/lone-faerie/mqttop/internal/secrets"
	"github.com/lone-faerie/mqttop/log"
)

type Config struct {
	Interval  time.Duration   `yaml:"interval"`
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

	FormatSize func(v int, bits bool) string `yaml:"-"`
}

var defaultCfg = &Config{
	Interval:  2 * time.Second,
	MQTT:      defaultMQTT,
	Discovery: defaultDiscovery,
	CPU:       defaultCPU,
	Memory:    defaultMemory,
	Disks:     defaultDisks,
	Net:       defaultNet,
	Battery:   defaultBattery,
	GPU:       defaultGPU,
}

func Default() *Config {
	cfg := defaultCfg
	cfg.load()
	return cfg
}

func Load(file ...string) (cfg *Config, err error) {
	log.Info("Loading config", "path", file)
	cfg = defaultCfg
	if _, err = os.Stat(file[0]); err != nil {
		return
	}
	r := byteutil.NewMultiFileReader(file...)
	defer r.Close()
	if err = yaml.NewDecoder(r).Decode(cfg); err != nil {
		return
	}
	err = cfg.load()
	log.Info("Config loaded")
	return
}

func (cfg *Config) loadValue(v reflect.Value) error {
	iface := v.Addr().Interface()
	if l, ok := iface.(loader); ok {
		return l.load(cfg)
	}
	return nil
}

func (cfg *Config) load() (err error) {
	var (
		v = reflect.ValueOf(cfg).Elem()
		n = v.NumField()
	)
	expand(v)
	for i := 0; i < n; i++ {
		f := v.Field(i)
		if f.Kind() != reflect.Slice {
			if err = cfg.loadValue(f); err != nil {
				return
			}
			continue
		}
		for j := 0; j < f.Len(); j++ {
			if err = cfg.loadValue(f.Index(j)); err != nil {
				return
			}
		}
	}
	return
}

func expand(v reflect.Value) {
	switch v.Kind() {
	case reflect.String:
		s := v.String()
		if secret, ok := secrets.CutPrefix(s); ok {
			s = secrets.MustRead(secret)
		} else {
			s = os.ExpandEnv(s)
		}
		v.SetString(s)
	case reflect.Struct:
		n := v.NumField()
		for i := 0; i < n; i++ {
			expand(v.Field(i))
		}
	case reflect.Slice, reflect.Array:
		n := v.Len()
		for i := 0; i < n; i++ {
			expand(v.Index(i))
		}

	case reflect.Pointer:
		expand(v.Elem())
	}
}

func (cfg *Config) Expand() {
	v := reflect.ValueOf(cfg).Elem()
	expand(v)
}

func (cfg *Config) Write(file string) error {
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := yaml.NewEncoder(f)
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

func (cfg *Config) SetInterval(d time.Duration) {
	setInterval(reflect.ValueOf(cfg).Elem(), d)
}

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
