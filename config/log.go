package config

import "github.com/lone-faerie/mqttop/log"

// LogConfig is the configuration for logging.
type LogConfig struct {
	// Level is the minimum level used for logging.
	Level log.Level `yaml:"level"`
	// Output is the location logs should be output to.
	// Acceptable values are either a path to a file
	// or one of the following special values:
	// - "stderr" (default)
	// - "stdout"
	Output string `yaml:"output"`
	// Format is the format used for logging. If blank then the
	// default format is used. The acceptable values are:
	// - "json"
	// - "text"
	Format string `yaml:"format"`
}
