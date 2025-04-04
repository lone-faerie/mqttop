// Package mqttop implements a bridge to provide system metrics to the MQTT broker.
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
//
// Full documentation is available at:
// https://pkg.go.dev/github.com/lone-faerie/mqttop
package mqttop
