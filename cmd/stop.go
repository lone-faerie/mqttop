package main

import (
	"errors"
	"os"
	"os/exec"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/cobra"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
)

var StopCommand = &cobra.Command{
	Use:   "stop",
	Short: "Stop running bridge",
	Args:  cobra.MaximumNArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) (err error) {
		log.SetLogLevel(log.LevelWarn)
		initConfig()
		cfg, err = config.Load(ConfigPath...)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return
		}
		if err = flagsToConfig(cfg, cmd, args); err != nil {
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

func init() {
	StopCommand.Flags().StringSliceVarP(&ConfigPath, "config", "c", nil, "Path(s) to config file/directory")
	StopCommand.Flags().StringVarP(&Broker, "broker", "b", "", "MQTT broker address")
	StopCommand.Flags().IntVarP(&Port, "port", "p", 1883, "MQTT broker port")
	StopCommand.Flags().StringVar(&Username, "username", "", "MQTT client username")
	StopCommand.Flags().StringVar(&Password, "password", "", "MQTT client password")
	StopCommand.Flags().IntP("pid", "P", 0, "PID of the process")

	RootCommand.AddCommand(StopCommand)
}
