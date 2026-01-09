package queue

import (
	"encoding/json"
	"fmt"
	"github.com/XANi/promwriter"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Queue struct {
	client    mqtt.Client
	sensorMap map[string]Sensor
	sync.RWMutex
}

type Config struct {
	MQTTAddr    string
	Logger      *zap.SugaredLogger
	ExtraLabels map[string]string
}

func New(cfg *Config) (*Queue, error) {
	mqttURL, err := url.Parse(cfg.MQTTAddr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse MQTT URL: %w", err)
	}
	p, _ := mqttURL.User.Password()
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTAddr).
		SetUsername(mqttURL.User.Username()).
		SetPassword(p).
		SetClientID("esphome2prom").
		SetKeepAlive(2 * time.Second).
		SetPingTimeout(1 * time.Second)

	client := mqtt.NewClient(opts)
	q := &Queue{
		client:    client,
		sensorMap: map[string]Sensor{},
	}
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//token := client.Publish("esphome/discover", 0, false, "hello mqtt")
	//token.Wait()
	promcfg := promwriter.Config{
		URL:              os.Getenv("PROMETHEUS_WRITE_URL"),
		MaxBatchDuration: time.Second * 1,
		MaxBatchLength:   10,
		Logger:           cfg.Logger.Named("promwriter"),
	}
	pw, err := promwriter.New(promcfg)
	if err != nil {
		cfg.Logger.Panicw("promwriter", "err", err)
	}
	sendQueue := make(chan Metric, 128)
	go func() {
		for ev := range sendQueue {
			metric := promwriter.Metric{
				Name:   ev.Name,
				Labels: ev.Labels,
				TS:     ev.TS.UTC(),
				Value:  ev.Value,
			}
			for k, v := range cfg.ExtraLabels {
				ev.Labels[k] = v
			}
			err := pw.WriteMetric(metric)
			if err != nil {
				cfg.Logger.Warnf("error writing metric %+v: %s", metric, err)
			}
		}
	}()
	client.Subscribe("homeassistant/#", 0, func(c mqtt.Client, m mqtt.Message) {
		d := ESPHomeDiscovery{}
		err := json.Unmarshal(m.Payload(), &d)
		if err != nil {
			cfg.Logger.Warnf("could not decode discovery %s: %s\n", m.Topic(), string(m.Payload()))
			return
		}
		cfg.Logger.Debugf("received %s: %+v\n", m.Topic(), &d)
		if d.StateTopic != "" {
			switch d.DeviceClass {
			case DeviceClassTemperature:
				cfg.Logger.Infof("adding temperature sensor under %s", d.StateTopic)
				sensor := NewTemperatureSensor(cfg.Logger.Named(m.Topic()), d, sendQueue)
				q.Lock()
				q.sensorMap[d.StateTopic] = sensor
				q.Unlock()
			default:
				cfg.Logger.Infof("[%s] unknown device class [%s]", m.Topic(), d.DeviceClass)
			}
		}
	})
	client.Subscribe("+/sensor/#", 0, func(c mqtt.Client, m mqtt.Message) {
		if strings.HasPrefix(m.Topic(), "homeassistant/") {
			return
		}
		q.RLock() // optimize that lock out
		if f, ok := q.sensorMap[m.Topic()]; ok {
			//cfg.Logger.Debugf("%s: %s", m.Topic(), string(m.Payload()))
			err := f.ProcessMessage(m)
			if err != nil {
				cfg.Logger.Warnf("could not process message %s: %s\n", m.Topic(), string(m.Payload()))
			}
		} else {
			cfg.Logger.Warnf("err: %s: %s", m.Topic(), string(m.Payload()))
		}
		q.RUnlock()
	})
	return q, nil
}
