package cmd

import (
	"errors"
	"os"
	"os/exec"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
)

// Usage:
//   mqttop stop [flags]
//
// Flags:
//   -b, --broker string     MQTT broker address
//   -c, --config strings    Path(s) to config file/directory
//   -h, --help              help for stop
//       --password string   MQTT client password
//   -P, --pid int           PID of the process
//   -p, --port int          MQTT broker port (default 1883)
//       --username string   MQTT client username
func NewCmdStop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop running bridge",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			log.SetLogLevel(log.LevelWarn)
			findConfig()
			cfg, err = config.Load(ConfigPath...)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return
			}
			if err = flagsToConfig(cfg, args); err != nil {
				return
			}
			log.Info("Config loaded")
			setLogHandler(cfg, log.LevelWarn)
			log.Debug("MQTT broker", "addr", cfg.MQTT.Broker)
			return
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if pid := cmd.Flags().Lookup("pid"); pid != nil && pid.Changed && pid.Value.String() != pid.DefValue {
				c := "ps cax | grep -qe '" + pid.Value.String() + "[[:space:]].*mqttop' && kill -2 " + pid.Value.String()
				log.Debug("Stopping", "pid", pid.Value)
				if err := exec.Command("sh", "-c", c).Run(); err == nil {
					return nil
				}
			}
			opts := cfg.MQTT.ClientOptions()
			client := mqtt.NewClient(opts)
			t := client.Connect()
			t.Wait()
			if err := t.Error(); err != nil {
				return err
			}
			defer client.Disconnect(500)
			var topic string
			if len(args) > 0 {
				topic = args[1]
			} else {
				topic = "mqttop/bridge/stop"
			}
			t = client.Publish(topic, 0, false, []byte{})
			t.Wait()
			return t.Error()
		},
	}

	cmd.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	cmd.Flags().StringVarP(&Broker, "broker", "b", "", "MQTT broker address")
	cmd.Flags().IntVarP(&Port, "port", "p", 1883, "MQTT broker port")
	cmd.Flags().StringVar(&Username, "username", "", "MQTT client username")
	cmd.Flags().StringVar(&Password, "password", "", "MQTT client password")
	cmd.Flags().IntP("pid", "P", 0, "PID of the process")

	cmd.SetHelpTemplate(cmd.HelpTemplate() + "\n" + fullDocsFooter + "\n")

	return cmd
}
