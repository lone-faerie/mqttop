package mqttop

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"

	"github.com/eclipse/paho.mqtt.golang"
)

type MockClient struct {
	connected bool
	onConnect mqtt.OnConnectHandler
	w         io.Writer
	mu        sync.Mutex
}

func NewMockClient(o *mqtt.ClientOptions, discard bool) mqtt.Client {
	c := &MockClient{
		onConnect: o.OnConnect,
		w:         os.Stdout,
	}
	if discard {
		c.w = io.Discard
	}
	return c
}

func (c *MockClient) IsConnected() bool {
	return c.connected
}

func (c *MockClient) IsConnectionOpen() bool {
	return c.connected
}

func (c *MockClient) Connect() mqtt.Token {
	c.connected = true
	if c.onConnect != nil {
		c.onConnect(c)
	}
	return &mqtt.DummyToken{}
}

func (c *MockClient) Disconnect(_ uint) {
	c.connected = false
}

type syncer interface {
	Sync() error
}

func (c *MockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, _ := payload.([]byte)
	e := json.NewEncoder(c.w)
	e.SetIndent("", "  ")
	err := e.Encode(map[string]json.RawMessage{topic: json.RawMessage(p)})
	if err != nil {
		log.Println("Error encoding", topic, err)
	}
	c.w.Write([]byte{'\n', '\n'})
	if s, ok := c.w.(syncer); ok {
		s.Sync()
	}
	return &mqtt.DummyToken{}
}

func (c *MockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mqtt.DummyToken{}
}

func (c *MockClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	return &mqtt.DummyToken{}
}

func (c *MockClient) Unsubscribe(topics ...string) mqtt.Token {
	return &mqtt.DummyToken{}
}

func (c *MockClient) AddRoute(topic string, callback mqtt.MessageHandler) {}

func (c *MockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(&mqtt.ClientOptions{})
}
