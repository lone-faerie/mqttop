package mock

import (
	"encoding/json"
	"io"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/lone-faerie/mqttop/log"
)

type MockClient struct {
	connected bool

	onConnect mqtt.OnConnectHandler
	msg       []byte
	opts      *mqtt.ClientOptions
	w         io.Writer
	mu        sync.Mutex
}

func NewMockClient(o *mqtt.ClientOptions, w io.Writer) mqtt.Client {
	c := &MockClient{
		opts: o,
		w:    w,
	}
	return c
}

func (c *MockClient) SetCallbackMessage(msg []byte) {
	c.msg = msg
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

func (c *MockClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.mu.Lock()
	defer c.mu.Unlock()
	var p json.RawMessage
	switch v := payload.(type) {
	case []byte:
		p = json.RawMessage(v)
	case string:
		p = json.RawMessage(v)
	}
	e := json.NewEncoder(c.w)
	e.SetIndent("", "  ")
	err := e.Encode(map[string]json.RawMessage{topic: json.RawMessage(p)})
	if err != nil {
		log.Error("Error encoding "+topic, err)
	}
	//c.w.Write([]byte{'\n', '\n'})
	if s, ok := c.w.(interface{ Sync() error }); ok {
		s.Sync()
	}
	return &mqtt.DummyToken{}
}

func (c *MockClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	callback(c, &message{topic: topic, payload: c.msg})
	return &mqtt.DummyToken{}
}

func (c *MockClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic := range filters {
		callback(c, &message{topic: topic, payload: c.msg})
	}
	return &mqtt.DummyToken{}
}

func (c *MockClient) Unsubscribe(topics ...string) mqtt.Token {
	return &mqtt.DummyToken{}
}

func (c *MockClient) AddRoute(topic string, callback mqtt.MessageHandler) {}

func (c *MockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(c.opts)
}

type message struct {
	topic   string
	payload []byte
}

func (m *message) Duplicate() bool   { return false }
func (m *message) Qos() byte         { return 0 }
func (m *message) Retained() bool    { return false }
func (m *message) MessageID() uint16 { return 0 }
func (m *message) Ack()              {}

func (m *message) Topic() string {
	return m.topic
}

func (m *message) Payload() []byte {
	return m.payload
}
