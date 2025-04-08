package mqttop

import (
	"context"
	"io"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/lone-faerie/mqttop/config"
	"github.com/lone-faerie/mqttop/log"
	"github.com/lone-faerie/mqttop/mock"
)

func mockBridge(w io.Writer) *Bridge {
	cfg := config.Default()
	opts := mqtt.NewClientOptions()
	if w == nil {
		w = io.Discard
	}
	client := mock.NewMockClient(opts, w)
	return NewWithClient(cfg, client)
}

func TestBridgeConnect(t *testing.T) {
	bridge := mockBridge(nil)
	ctx := context.Background()
	if err := bridge.Connect(ctx); err != nil {
		t.Error(err)
	}
	t.Cleanup(bridge.Disconnect)
}

func TestMain(m *testing.M) {
	log.SetLogLevel(log.LevelDisabled)
	m.Run()
}
