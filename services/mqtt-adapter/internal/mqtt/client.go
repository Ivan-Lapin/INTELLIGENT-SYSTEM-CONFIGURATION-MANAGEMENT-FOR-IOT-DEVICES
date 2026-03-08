package mqtt

import (
	"encoding/json"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
)

type Client struct {
	c paho.Client
}

type Config struct {
	BrokerURL string
	ClientID  string
}

func New(cfg Config) (*Client, error) {
	opts := paho.NewClientOptions().
		AddBroker(cfg.BrokerURL).
		SetClientID(cfg.ClientID).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(true)

	c := paho.NewClient(opts)
	tok := c.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		return nil, tok.Error()
	}
	if tok.Error() != nil {
		return nil, tok.Error()
	}
	return &Client{c: c}, nil
}

func (m *Client) PublishJSON(topic string, qos byte, retain bool, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tok := m.c.Publish(topic, qos, retain, b)
	tok.Wait()
	return tok.Error()
}

func (m *Client) Subscribe(topic string, qos byte, handler paho.MessageHandler) error {
	tok := m.c.Subscribe(topic, qos, handler)
	tok.Wait()
	return tok.Error()
}

func (m *Client) Close() {
	if m.c != nil && m.c.IsConnected() {
		m.c.Disconnect(250)
	}
	log.Println("mqtt disconnected")
}
