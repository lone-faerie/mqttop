package config

import (
	"crypto/tls"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/log"
)

type MQTTConfig struct {
	Broker            string        `yaml:"broker"`
	ClientID          string        `yaml:"client_id,omitempty"`
	Username          string        `yaml:"username"`
	Password          string        `yaml:"password"`
	KeepAlive         time.Duration `yaml:"keep_alive,omitempty"`
	CertFile          string        `yaml:"cert_file,omitempty"`
	KeyFile           string        `yaml:"key_file,omitempty"`
	ReconnectInterval time.Duration `yaml:"reconnect_interval,omitempty"`
	ConnectTimeout    time.Duration `yaml:"connect_timeout,omitempty"`
	PingTimeout       time.Duration `yaml:"ping_timeout,omitempty"`
	WriteTimeout      time.Duration `yaml:"write_timeout,omitempty"`
	BirthWillEnabled  bool          `yaml:"birth_lwt_enabled"`
	BirthWillTopic    string        `yaml:"birth_lwt_topic"`
	LogLevel          log.Level     `yaml:"log_level"`

	tlsCert *tls.Certificate
}

type DiscoveryConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Prefix       string `yaml:"prefix"`
	DeviceName   string `yaml:"device_name,omitempty"`
	NodeID       string `yaml:"node_id,omitempty"`
	Availability string `yaml:"availability_topic,omitempty"`
	Retained     bool   `yaml:"retained"`
	QoS          byte   `yaml:"qos,omitempty"`
	WaitTopic    string `yaml:"wait_topic"`
	WaitPayload  string `yaml:"wait_payload"`
}

var defaultMQTT = MQTTConfig{
	Broker:           "$MQTTOP_BROKER_ADDRESS",
	Username:         "$MQTTOP_BROKER_USERNAME",
	Password:         "$MQTTOP_BROKER_PASSWORD",
	BirthWillEnabled: true,
	BirthWillTopic:   "mqttop/bridge/status",
	LogLevel:         log.LevelDisabled,
}

var defaultDiscovery = DiscoveryConfig{
	Enabled:  true,
	Prefix:   "homeassistant",
	Retained: true,
}

func (cfg *MQTTConfig) ClientOptions() *mqtt.ClientOptions {
	o := mqtt.NewClientOptions()
	o.AddBroker(cfg.Broker)
	o.SetClientID(cfg.ClientID)
	o.SetUsername(cfg.Username).SetPassword(cfg.Password)
	if cfg.KeepAlive > 0 {
		o.SetKeepAlive(cfg.KeepAlive)
	}
	if cfg.ReconnectInterval > 0 {
		o.SetMaxReconnectInterval(cfg.ReconnectInterval)
	}
	if cfg.ConnectTimeout > 0 {
		o.SetConnectTimeout(cfg.ConnectTimeout)
	}
	if cfg.PingTimeout > 0 {
		o.SetPingTimeout(cfg.PingTimeout)
	}
	if cfg.WriteTimeout > 0 {
		o.SetWriteTimeout(cfg.WriteTimeout)
	}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		o.SetTLSConfig(&tls.Config{
			GetCertificate: cfg.getCertificate,
		})
	}
	return o
}

func (cfg *MQTTConfig) getCertificate(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if cfg.tlsCert == nil {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}
		cfg.tlsCert = &cert
	}
	return cfg.tlsCert, nil
}

func (cfg MQTTConfig) IsZero() bool {
	return cfg == defaultMQTT
}

func (cfg DiscoveryConfig) IsZero() bool {
	return cfg == defaultDiscovery
}
