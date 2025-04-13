package config_test

import (
	"os"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/config"
)

func TestReplaceBase(t *testing.T) {
	var tests = []struct {
		base  string
		topic string
		want  string
	}{
		{"base", "~/topic/foo", "base/topic/foo"},
		{"base", "topic/foo/~", "topic/foo/base"},
		{"base", "~/topic/foo/~", "base/topic/foo/base"},
		{"base", "topic/~/foo", "topic/~/foo"},
	}
	for _, tt := range tests {
		got := config.ReplaceBase(tt.base, tt.topic)
		if got != tt.want {
			t.Errorf("%q: wanted %q, got %q", tt.topic, tt.want, got)
		}
	}
}

func TestExpand(t *testing.T) {
	var tests = []struct {
		name   string
		value  string
		input  string
		want   string
		secret bool
		fail   bool
	}{
		{"foo", "Hello", "!secret foo", "Hello", true, false},
		{"bar", "World", "!secret bar", "World", true, false},
		{"BAZ", "env variable", "$BAZ", "env variable", false, false},
		{"", "", "$NOT_A_VAR", "", false, true},
	}

	t.Cleanup(func() {
		for _, tt := range tests {
			if tt.secret {
				os.Remove("/run/secrets/" + tt.name)
			}
		}
	})

	for _, tt := range tests {
		if tt.fail {
			continue
		}
		if tt.secret {
			err := os.WriteFile("/run/secrets/"+tt.name, []byte(tt.value), 0666)
			if os.IsPermission(err) {
				t.Skip("Skipping expand:", err)
			} else if err != nil {
				t.Fatalf("Error writing /run/secrets/%s: %v", tt.name, err)
			}
		} else {
			t.Setenv(tt.name, tt.value)
		}
	}

	for _, tt := range tests {
		got := config.Expand(tt.input)
		if got != tt.want {
			t.Errorf("%q: wanted %q, got %q", tt.input, tt.want, got)
		}
	}
}

func TestConfigSetInterval(t *testing.T) {
	cfg := config.Default()
	cfg.SetInterval(time.Minute)

	if cfg.Interval != time.Minute {
		t.Errorf("cfg.Interval: wanted %v, got %v", time.Minute, cfg.Interval)
	}
	if cfg.CPU.Interval != time.Minute {
		t.Errorf("cfg.CPU.Interval: wanted %v, got %v", time.Minute, cfg.CPU.Interval)
	}
	if cfg.Memory.Interval != time.Minute {
		t.Errorf("cfg.Memory.Interval: wanted %v, got %v", time.Minute, cfg.Memory.Interval)
	}
	if cfg.Disks.Interval != time.Minute {
		t.Errorf("cfg.Disks.Interval: wanted %v, got %v", time.Minute, cfg.Disks.Interval)
	}
	if cfg.Net.Interval != time.Minute {
		t.Errorf("cfg.Net.Interval: wanted %v, got %v", time.Minute, cfg.Net.Interval)
	}
	if cfg.Battery.Interval != time.Minute {
		t.Errorf("cfg.Battery.Interval: wanted %v, got %v", time.Minute, cfg.Battery.Interval)
	}
	if cfg.GPU.Interval != time.Minute {
		t.Errorf("cfg.GPU.Interval: wanted %v, got %v", time.Minute, cfg.GPU.Interval)
	}
}

func TestConfigSetMetrics(t *testing.T) {
	cfg := config.Default()
	cfg.SetMetrics("battery", "disks")

	if cfg.CPU.Enabled {
		t.Error("cfg.CPU.Enabled: wanted false, got true")
	}
	if cfg.Memory.Enabled {
		t.Error("cfg.Memory.Enabled: wanted false, got true")
	}
	if !cfg.Disks.Enabled {
		t.Error("cfg.Disks.Enabled: wanted true, got false")
	}
	if cfg.Net.Enabled {
		t.Error("cfg.Net.Enabled: wanted false, got true")
	}
	if !cfg.Battery.Enabled {
		t.Error("cfg.Battery.Enabled: wanted true, got false")
	}
	if cfg.GPU.Enabled {
		t.Error("cfg.GPU.Enabled: wanted false, got true")
	}

	t.Run("All", func(t *testing.T) {
		cfg.SetMetrics("all")

		if !cfg.CPU.Enabled {
			t.Error("cfg.CPU.Enabled: wanted true, got false")
		}
		if !cfg.Memory.Enabled {
			t.Error("cfg.Memory.Enabled: wanted true, got false")
		}
		if !cfg.Disks.Enabled {
			t.Error("cfg.Disks.Enabled: wanted true, got false")
		}
		if !cfg.Net.Enabled {
			t.Error("cfg.Net.Enabled: wanted true, got false")
		}
		if !cfg.Battery.Enabled {
			t.Error("cfg.Battery.Enabled: wanted true, got false")
		}
		if !cfg.GPU.Enabled {
			t.Error("cfg.GPU.Enabled: wanted true, got false")
		}
	})
}

func TestParseRescan(t *testing.T) {
	var tests = []struct {
		rescan   string
		interval time.Duration
		want     time.Duration
	}{
		{"true", time.Second, time.Second},
		{"false", time.Second, 0},
		{"3s", time.Second, 3 * time.Second},
		{"foo", time.Second, 0},
	}

	for _, tt := range tests {
		var got time.Duration
		switch tt.rescan {
		case "true", "True", "TRUE", "y", "Y", "yes", "Yes", "YES", "on", "On", "ON":
			got = tt.interval
		case "false", "False", "FALSE", "n", "N", "no", "No", "NO", "off", "Off", "OFF":
		case "":
		default:
			got, _ = time.ParseDuration(tt.rescan)
		}
		if got != tt.want {
			t.Errorf("%q: wanted %v, got %v", tt.rescan, tt.want, got)
		}
	}
}

func TestMQTTClientOptions(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		cfg := config.Default()
		got := cfg.MQTT.ClientOptions()
		want := mqtt.NewClientOptions()

		if got.ClientID != want.ClientID {
			t.Errorf("ClientID: wanted %q, got %q", want.ClientID, got.ClientID)
		}
		if got.CleanSession != want.CleanSession {
			t.Errorf("CleanSession: wanted %v, got %v", want.CleanSession, got.CleanSession)
		}
		if got.Order != want.Order {
			t.Errorf("Order: wanted %v, got %v", want.Order, got.Order)
		}
		if got.KeepAlive != want.KeepAlive {
			t.Errorf("KeepAlive: wanted %v, got %v", want.KeepAlive, got.KeepAlive)
		}
		if got.ConnectTimeout != want.ConnectTimeout {
			t.Errorf("ConnectTimeout: wanted %v, got %v", want.ConnectTimeout, got.ConnectTimeout)
		}
		if got.MaxReconnectInterval != want.MaxReconnectInterval {
			t.Errorf("MaxReconnectInterval: wanted %v, got %v", want.MaxReconnectInterval, got.MaxReconnectInterval)
		}
		if got.AutoReconnect != want.AutoReconnect {
			t.Errorf("AutoReconnect: wanted %v, got %v", want.AutoReconnect, got.AutoReconnect)
		}
		if got.PingTimeout != want.PingTimeout {
			t.Errorf("PingTimeout: wanted %v, got %v", want.PingTimeout, got.PingTimeout)
		}
		if got.TLSConfig != want.TLSConfig {
			t.Errorf("TLSConfig: wanted %v, got %v", want.TLSConfig, got.TLSConfig)
		}
		if !got.ResumeSubs {
			t.Errorf("ResumeSubs: wanted true, got false")
		}
	})

	t.Run("Modify", func(t *testing.T) {
		cfg := config.Default()
		cfg.MQTT.ClientID = "foo"
		cfg.MQTT.ConnectTimeout = time.Second
		cfg.MQTT.CertFile = "cert.pem"
		cfg.MQTT.KeyFile = "key.pem"
		got := cfg.MQTT.ClientOptions()
		want := mqtt.NewClientOptions()
		want.SetClientID("foo")
		want.SetConnectTimeout(time.Second)

		if got.ClientID != want.ClientID {
			t.Errorf("ClientID: wanted %q, got %q", want.ClientID, got.ClientID)
		}
		if got.CleanSession != want.CleanSession {
			t.Errorf("CleanSession: wanted %v, got %v", want.CleanSession, got.CleanSession)
		}
		if got.Order != want.Order {
			t.Errorf("Order: wanted %v, got %v", want.Order, got.Order)
		}
		if got.KeepAlive != want.KeepAlive {
			t.Errorf("KeepAlive: wanted %v, got %v", want.KeepAlive, got.KeepAlive)
		}
		if got.ConnectTimeout != want.ConnectTimeout {
			t.Errorf("ConnectTimeout: wanted %v, got %v", want.ConnectTimeout, got.ConnectTimeout)
		}
		if got.MaxReconnectInterval != want.MaxReconnectInterval {
			t.Errorf("MaxReconnectInterval: wanted %v, got %v", want.MaxReconnectInterval, got.MaxReconnectInterval)
		}
		if got.AutoReconnect != want.AutoReconnect {
			t.Errorf("AutoReconnect: wanted %v, got %v", want.AutoReconnect, got.AutoReconnect)
		}
		if got.PingTimeout != want.PingTimeout {
			t.Errorf("PingTimeout: wanted %v, got %v", want.PingTimeout, got.PingTimeout)
		}
		if got.TLSConfig == nil {
			t.Errorf("TLSConfig: got %v", got.TLSConfig)
		}
		if !got.ResumeSubs {
			t.Errorf("ResumeSubs: wanted true, got false")
		}
	})
}
