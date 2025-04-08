// Package mqttop implements a bridge to provide system metrics to the MQTT broker.
//
// It is primarily meant to be run from the command line, but may also be used as library.
//
// # Command line
//
// Run a bridge to provide system metrics to the MQTT broker.
//
// A connection to the MQTT broker will be established and the bridge will run in the foreground until a signal is received.
//
//   - SIGINT or SIGTERM will gracefully shutdown the bridge.
//
// MQTTop can load configuration from multiple YAML files, including from directories. If no config file is specified, the default path(s) will be determined by the first defined value of $MQTTOP_CONFIG_PATH, $XDG_CONFIG_HOME/mqttop.yaml, or $HOME/.config/mqttop.yaml. In the case of $MQTTOP_CONFIG_PATH, the value may be a comma-separated list of paths. If none of these files exist, the default configuration will be used, which looks for the following environment variables:
//
//   - broker:   $MQTTOP_BROKER_ADDRESS
//   - username: $MQTTOP_BROKER_USERNAME
//   - password: $MQTTOP_BROKER_PASSWORD
//
// Enabled metrics may be supplied as arguments, which will ignore the enabled metrics of the config. The special argument 'all' may be supplied to enable all metrics. The valid arguments include:
//
//   - all, cpu, memory, disks, net, battery, dirs, gpu
//
// All of the flags, if specified, will override the equivalent values in the config. The format of --broker should be scheme://host:port Where "scheme" is one of "tcp", "ssl", or "ws", "host" is the ip-address (or hostname) and "port" is the port on which the broker is accepting connections. If "scheme" is not defined, it defaults to "tcp" and if "port" is not defined, it will use the value of --port (default 1883).
//
// Usage:
//
//	mqttop run [--config <path>]... [flags] [metric]...
//
// Aliases:
//
//	run, start
//
// Examples:
//
//	mqttop run --config config.yaml
//	mqttop run --config config.yaml cpu memory
//	mqttop run --broker 127.0.0.1:1883 --username mqttop --password p@55w0rd
//
// Flags:
//
//	-c, --config strings      Path(s) to config file/directory
//	-b, --broker string       MQTT broker address
//	-p, --port int            MQTT broker port (default 1883)
//	    --username string     MQTT client username
//	    --password string     MQTT client password
//	    --cert string         MQTT TLS certificate file (PEM encoded)
//	    --key string          MQTT TLS private key file (PEM encoded)
//	-i, --interval duration   Update interval
//	-D, --discovery string    Discovery prefix, or 'disabled' to disable
//	-l, --log string          Log level
//	-d, --detach              Run detached (in background)
//	-h, --help                help for run
//
// # Package
//
//	import (
//		"context"
//		"log"
//		"os"
//		"os/signal"
//
//		"github.com/lone-faerie/mqttop"
//		"github.com/lone-faerie/mqttop/config"
//	)
//
//	func main() {
//		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
//		defer cancel()
//
//		cfg := config.Default()
//		bridge := mqttop.New(cfg)
//
//		if err := bridge.Connect(ctx); err != nil {
//			log.Fatal(err)
//		}
//		defer bridge.Disconnect()
//
//		bridge.Start(ctx)
//		select {
//		case <-ctx.Done():
//			return
//		case err := <-bridge.Ready():
//			if err != nil {
//				log.Fatal(err)
//			}
//		}
//
//		select {
//		case <-ctx.Done():
//		case <-bridge.Done():
//		}
//	}
//
// Full documentation is available at:
// https://pkg.go.dev/github.com/lone-faerie/mqttop
package mqttop
