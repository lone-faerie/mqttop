package config

import "github.com/lone-faerie/mqttop/log"

type LogConfig struct {
	Level  log.Level `yaml:"level"`
	Output string    `yaml:"output"`
	Format string    `yaml:"format"`
}
