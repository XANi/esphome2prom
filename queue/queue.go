package queue

import (
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"time"
)

type Queue struct {
	Client mqtt.Client
}

type Config struct {
	MQTTAddr string
	Logger   *zap.SugaredLogger
}

func New(cfg *Config) (*Queue, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTAddr).
		SetUsername("mqtt").
		SetPassword("mqtt").
		SetClientID("esphome2prom").
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	q := &Queue{
		Client: client,
	}
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//token := client.Publish("esphome/discover", 0, false, "hello mqtt")
	//token.Wait()

	client.Subscribe("homeassistant/#", 0, func(c mqtt.Client, m mqtt.Message) {
		d := ESPHomeDiscovery{}
		err := json.Unmarshal(m.Payload(), &d)
		if err != nil {
			cfg.Logger.Warnf("could not decode discovery %s: %s\n", m.Topic(), string(m.Payload()))
			return
		}
		cfg.Logger.Infof("received %s: %+v\n", m.Topic(), &d)
	})
	return q, nil
}
