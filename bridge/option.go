package bridge

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lone-faerie/mqttop/discovery"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/metrics"
)

type Option func(*Bridge)

func WithClient(c mqtt.Client) Option {
	return func(b *Bridge) {
		b.client = c
	}
}

func WithDiscovery(d *discovery.Discovery, migrate bool) Option {
	return func(b *Bridge) {
		b.discovery = d
		b.migrate = migrate
	}
}

func WithMetrics(m ...metrics.Metric) Option {
	return func(b *Bridge) {
		b.metrics = append(b.metrics, m...)
	}
}

func WithLogLevel(level log.Level) Option {
	return func(b *Bridge) {
		if level <= log.LevelError {
			mqtt.ERROR = log.ErrorLogger()
			mqtt.CRITICAL = log.ErrorLogger()
			mqtt.WARN = log.WarnLogger()
			mqtt.DEBUG = log.DebugLogger()
		} else if level <= log.LevelWarn {
			mqtt.ERROR = mqtt.NOOPLogger{}
			mqtt.CRITICAL = mqtt.NOOPLogger{}
			mqtt.WARN = log.WarnLogger()
			mqtt.DEBUG = log.DebugLogger()
		} else if level <= log.LevelDebug {
			mqtt.ERROR = mqtt.NOOPLogger{}
			mqtt.CRITICAL = mqtt.NOOPLogger{}
			mqtt.WARN = mqtt.NOOPLogger{}
			mqtt.DEBUG = log.DebugLogger()
		} else {
			mqtt.ERROR = mqtt.NOOPLogger{}
			mqtt.CRITICAL = mqtt.NOOPLogger{}
			mqtt.WARN = mqtt.NOOPLogger{}
			mqtt.DEBUG = mqtt.NOOPLogger{}
		}
	}
}

func WithTopicPrefix(prefix string) Option {
	return func(b *Bridge) {
		b.topicPrefix = prefix
	}
}
