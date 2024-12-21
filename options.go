package mqtt

import (
	"fmt"
	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/requests"
)

type Listen struct {
	URL      string `yaml:"url"`
	CertFile string `yaml:"certFile"`
	KeyFile  string `yaml:"keyFile"`
}

type config struct {
	HTTP       Listen            `json:"HTTP"`
	MQTT       Listen            `json:"MQTT"`
	MQTTs      Listen            `json:"MQTTs"`
	WebSocket  Listen            `json:"Websocket"`
	WebSockets Listen            `json:"Websockets"`
	Auth       map[string]string `json:"Auth"`
}

func (c *config) GetAuth(username string) (string, bool) {
	password, ok := c.Auth[username]
	return password, ok
}

var CONFIG = &config{
	Auth: map[string]string{
		"":     "",
		"root": "admin",
	},
}

type Options struct {
	URL           string // client used
	ClientID      string
	Version       byte
	Subscriptions []packet.Subscription
}

type Option func(*Options)

func newOptions(opts ...Option) Options {
	options := Options{
		URL:      "mqtt://127.0.0.1:1883",
		ClientID: "mqtt-" + requests.GenId(),
		Version:  packet.VERSION311,
	}
	for _, o := range opts {
		o(&options)
	}
	return options
}

func URL(url string) Option {
	return func(o *Options) {
		o.URL = url
	}
}

func Subscription(subscription ...packet.Subscription) Option {
	return func(o *Options) {
		o.Subscriptions = append(o.Subscriptions, subscription...)
	}
}

func Version[T ~string | ~byte](version T) Option {
	return func(o *Options) {
		switch v := any(version).(type) {
		case byte:
			o.Version = v
		case string:
			switch v {
			case "5.0.0":
				o.Version = packet.VERSION500
			case "3.1.1":
				o.Version = packet.VERSION311
			default:
				panic(fmt.Errorf("version = %s not support", v))
			}
		}
	}
}
