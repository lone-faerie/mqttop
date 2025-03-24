//go:build !nvidia

package metrics

import "github.com/lone-faerie/mqttop/config"

func appendGPU(m []Metric, _ *config.Config) []Metric { return m }
