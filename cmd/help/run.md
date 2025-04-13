Run a bridge to provide system metrics to the MQTT broker.

A connection to the MQTT broker will be established and the bridge will run in the foreground until a signal is received.

	- SIGINT or SIGTERM will gracefully shutdown the bridge.

MQTTop can load configuration from multiple YAML files, including from directories. If no config file is specified, the default path(s) will be determined by the first defined value of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, or $HOME/.config/mqttop.yaml. In the case of $MQTTOP_CONFIG_PATH, the value may be a comma-separated list of paths. If none of these files exist, the default configuration will be used, which looks for the following environment variables:

	- broker:   $MQTTOP_BROKER_ADDRESS
	- username: $MQTTOP_BROKER_USERNAME
	- password: $MQTTOP_BROKER_PASSWORD

Enabled metrics may be supplied as arguments, which will ignore the enabled metrics of the config. The special argument 'all' may be supplied to enable all metrics. The valid arguments include:

	- all, cpu, memory, disks, net, battery, dirs, gpu

All of the flags, if specified, will override the equivalent values in the config. The format of --broker should be scheme://host:port Where "scheme" is one of "tcp", "ssl", or "ws", "host" is the ip-address (or hostname) and "port" is the port on which the broker is accepting connections. If "scheme" is not defined, it defaults to "tcp" and if "port" is not defined, it will use the value of --port (default 1883).
