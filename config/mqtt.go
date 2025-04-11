package config

import (
	"crypto/tls"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/log"
)

// MQTTConfig is the configuration for the MQTT client.
//
// See [mqtt.ClientOptions]
type MQTTConfig struct {
	// Broker is the URI of the broker. The format should be scheme://host:port
	// where "scheme" is one of "tcp", "ssl", or "ws", "host" is the ip-address
	// (or hostname) and "port" is the port on which the broker is accepting
	// connections.
	Broker string `yaml:"broker"`
	// ClientID is the (optional) client ID used when connecting to the broker.
	ClientID string `yaml:"client_id,omitempty"`
	// Username is the username used when connecting to the broker.
	Username string `yaml:"username"`
	// Password is the password used when connecting to the broker.
	Password string `yaml:"password"`
	// KeepAlive is the duration that the client should wait before pinging the broker.
	// This allows the client to know the connection hasn't been lost.
	KeepAlive time.Duration `yaml:"keep_alive,omitempty"`
	// CertFile is the path to the PEM-encoded TLS certificate. If blank (default) then
	// TLS is not used between the client and the broker.
	CertFile string `yaml:"cert_file,omitempty"`
	// KeyFile is the path to the PEM-encoded TLS private key. If blank (default) then
	// TLS is not used between the client and the broker.
	KeyFile string `yaml:"key_file,omitempty"`
	// ReconnectInterval is the maximum duration that the client will wait between reconnection
	// attempts.
	ReconnectInterval time.Duration `yaml:"reconnect_interval,omitempty"`
	// ConnectTimeout is the duration that the client will wait when attempting to open a
	// connection to the broker before timing out. A duration of 0 means the client will
	// never time out.
	ConnectTimeout time.Duration `yaml:"connect_timeout,omitempty"`
	// PingTimeout is the duration that the client will wait after pinging the broker to
	// determine if the connection was lost.
	PingTimeout time.Duration `yaml:"ping_timeout,omitempty"`
	// WriteTimeout is the duration that the client will block for when publishing a message
	// before unblocking with a timeout error. A duration of 0 means the client will never
	// time out.
	WriteTimeout time.Duration `yaml:"write_timeout,omitempty"`
	// BirthWillEnabled indicates if the Birth and Last Will and Testament messages are enabled.
	BirthWillEnabled bool `yaml:"birth_lwt_enabled"`
	// BirthWillTopic is the topic to publish the Birth and Last Will and Testament messages to
	// if enabled. The default value is "mqttop/bridge/status"
	BirthWillTopic string `yaml:"birth_lwt_topic"`
	// LogLevel is the log level to provide to the backing MQTT client package.
	// See [mqtt.Logger]
	LogLevel log.Level `yaml:"log_level"`

	tlsCert *tls.Certificate
}

// DiscoveryConfig is the configuration for performing MQTT discovery.
//
// See https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery
type DiscoveryConfig struct {
	Enabled bool `yaml:"enabled"`
	// Prefix is the discovery_prefix part of the discovery topic
	// in the form <discovery_prefix>/<component>/[<node_id>/]<object_id>/config.
	// The default value is "homeassistant"
	Prefix string `yaml:"prefix"`
	// Method is the method used for discovery. The acceptable values are:
	//	- "device" (default)
	//	- "components"
	//	- "nodes" (or "metrics")
	// If Method is "device" then a single discovery payload will be used for all
	// the components. If Method is "components" then a separate discovery payload
	// will be used for each component. If Method is "nodes" or "metrics" then a
	// separate discovery payload will be used for all the components of each metric.
	Method string `yaml:"method"`
	// DeviceName is the name of the device used for discovery. The default value
	// is "MQTTop" and the special value "hostname" means the device name will be
	// the hostname of the system, as determined by the contents of /etc/hostname.
	DeviceName string `yaml:"device_name,omitempty"`
	// NodeID is the (optional) node_id part of the discovery topic in the form
	// <discovery_prefix>/<component>/[<node_id>/]<object_id>/config. It may only
	// consist of characters from [a-zA-Z0-9_-]. If Method is "nodes" or "metrics"
	// then the node_id part of the topic will be the value <node_id>_<metric_type>.
	NodeID string `yaml:"node_id,omitempty"`
	// Availability is the topic used for reporting component availability. The default
	// value is "mqttop/bridge/status"
	Availability string `yaml:"availability_topic,omitempty"`
	// Retained indicates if the discovery payload should be retained at the broker.
	// The default value is false
	Retained bool `yaml:"retained"`
	// QoS is the Quality of Service used for the discovery payload and defines the
	// delivery guarantee of the payload. The acceptable values are:
	// - 0 (at most once, default)
	// - 1 (at least once)
	// - 2 (exactly once)
	QoS byte `yaml:"qos,omitempty"`
	// WaitTopic is the (optional) topic to wait for a message on before performing
	// discovery. If blank (default) then discovery is performed without waiting.
	WaitTopic string `yaml:"wait_topic"`
	// WaitPayload is the (optional) payload to wait for on WaitTopic. If blank
	// then wait for any payload.
	WaitPayload string `yaml:"wait_payload"`
}

var DefaultMQTT = MQTTConfig{
	Broker:           "$MQTTOP_BROKER_ADDRESS",
	Username:         "$MQTTOP_BROKER_USERNAME",
	Password:         "$MQTTOP_BROKER_PASSWORD",
	BirthWillEnabled: true,
	BirthWillTopic:   "~/bridge/status",
	LogLevel:         log.LevelDisabled,
}

var DefaultDiscovery = DiscoveryConfig{
	Enabled:      true,
	Prefix:       "homeassistant",
	Method:       "device",
	Availability: "~/bridge/status",
	Retained:     false,
}

// ClientOptions returns cfg formatted as [mqtt.ClientOptions] to provide to
// the backing MQTT client when calling [mqtt.NewClient].
func (cfg *MQTTConfig) ClientOptions() *mqtt.ClientOptions {
	o := mqtt.NewClientOptions()
	o.AddBroker(cfg.Broker)
	o.SetClientID(cfg.ClientID)
	o.SetUsername(cfg.Username).SetPassword(cfg.Password)
	o.SetResumeSubs(true)

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

	if cfg.BirthWillEnabled {
		o.SetWill(cfg.BirthWillTopic, "offline", 1, true)
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

// IsZero indicates whether cfg is the default value.
func (cfg MQTTConfig) IsZero() bool {
	return cfg == DefaultMQTT
}

// IsZero indicates whether cfg is the default value.
func (cfg DiscoveryConfig) IsZero() bool {
	return cfg == DefaultDiscovery
}
