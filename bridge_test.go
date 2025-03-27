package mqttop

import (
	"context"
	"log"
	"time"

	"github.com/lone-faerie/mqttop/config"
)

func ExampleBridge() {
	cfg := config.Default()
	bridge := New(cfg)

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if err := bridge.Connect(ctx); err != nil {
		log.Fatal("Error connecting to broker", err)
	}
	defer func() {
		bridge.Disconnect()
	}()
	bridge.Start(ctx)
	<-bridge.Ready()
	if cfg.Discovery.Enabled {
		bridge.Discover(ctx)
	}
	<-ctx.Done()
}
