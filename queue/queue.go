package queue

import (
	"encoding/json"
	"fmt"
	"github.com/XANi/promwriter"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/k0kubun/pp/v3"
	"go.uber.org/zap"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Queue struct {
	client    mqtt.Client
	cfg       *Config
	l         *zap.SugaredLogger
	sensorMap map[string]Sensor
	sendQueue chan Metric
	sync.RWMutex
}

type Config struct {
	MQTTAddr    string
	Logger      *zap.SugaredLogger
	ExtraLabels map[string]string
	Prefix      string
	Debug       bool
}

func New(cfg *Config) (*Queue, error) {
	mqttURL, err := url.Parse(cfg.MQTTAddr)
	if err != nil {
		return nil, fmt.Errorf("cannot parse MQTT URL: %w", err)
	}
	q := &Queue{
		sensorMap: map[string]Sensor{},
		cfg:       cfg,
		sendQueue: make(chan Metric, 128),
		l:         cfg.Logger,
	}
	p, _ := mqttURL.User.Password()

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.MQTTAddr).
		SetUsername(mqttURL.User.Username()).
		SetPassword(p).
		SetClientID("esphome2prom" + randomString(32)). // this need to be unique, else we get disconnect
		SetKeepAlive(20 * time.Second).
		SetPingTimeout(10 * time.Second).
		SetConnectRetry(true).
		SetConnectRetryInterval(30 * time.Second).
		SetAutoReconnect(true).
		SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
			cfg.Logger.Warnf("reconnecting to MQ")
		}).SetConnectionLostHandler(func(client mqtt.Client, err error) {
		cfg.Logger.Warnf("connection to MQTT lost: %v", err)
	}).SetOnConnectHandler(func(client mqtt.Client) {
		cfg.Logger.Infof("connected to MQTT")
		q.addSubscriptions()
	})
	client := mqtt.NewClient(opts)
	q.client = client

	if token := client.Connect(); token.Wait() {
		if token.Error() != nil {
			cfg.Logger.Panicf("err connecting: %s", token.Error())
		} else {
			cfg.Logger.Infof("connected to mqtt %s:%s", mqttURL.Hostname(), mqttURL.Port())
		}
	}
	go func() {
		failCount := 0
		for {
			time.Sleep(30 * time.Second)
			if !client.IsConnected() {
				failCount++
				failCount++
			} else if failCount > 0 {
				failCount--
			}
			if failCount > 10 {
				log.Panic("not connected for a while, exiting")
			}
		}
	}()
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
	go func() {

		for ev := range q.sendQueue {
			metric := promwriter.Metric{
				Name:   cfg.Prefix + ev.Name,
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

	return q, nil
}

func (q *Queue) addSubscriptions() {
	q.l.Debugf("adding subscriptions")
	q.client.Subscribe("homeassistant/#", 0, func(c mqtt.Client, m mqtt.Message) {
		d := ESPHomeDiscovery{}
		// wildcard must be last character in the topic so we can't just do `homeassistant/#/config` here
		if !strings.HasSuffix(m.Topic(), "/config") {
			return
		}
		err := json.Unmarshal(m.Payload(), &d)
		if err != nil {
			q.cfg.Logger.Warnf("could not decode discovery %s: %s\n", m.Topic(), string(m.Payload()))
			return
		}
		if q.cfg.Debug {
			q.cfg.Logger.Debugf("received %s: %+v\n", m.Topic(), pp.Sprint(&d))
		}
		if d.Dev.Name == "ignoreme" {
			q.l.Infof("ignoring %s", m.Topic())
			return
		}
		// https://www.home-assistant.io/integrations/sensor/#device-class
		if d.StateTopic != "" {
			sensorNotFound := false
			q.Lock()
			switch d.DeviceClass {
			case DeviceClassTemperature:
				q.sensorMap[d.StateTopic] = NewTemperatureSensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassPressure:
				q.sensorMap[d.StateTopic] = NewPressureSensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassHumidity:
				q.sensorMap[d.StateTopic] = NewHumiditySensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassSignalStrength:
				q.sensorMap[d.StateTopic] = NewSignalStrengthSensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassVoltage:
				q.sensorMap[d.StateTopic] = NewVoltageSensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassCurrent:
				q.sensorMap[d.StateTopic] = NewCurrentSensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassCO2:
				q.sensorMap[d.StateTopic] = NewCO2Sensor(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassParticulate1:
				q.sensorMap[d.StateTopic] = NewParticulateSensor1(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassParticulate25:
				q.sensorMap[d.StateTopic] = NewParticulateSensor25(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassParticulate4:
				q.sensorMap[d.StateTopic] = NewParticulateSensor4(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassParticulate10:
				q.sensorMap[d.StateTopic] = NewParticulateSensor10(q.l.Named(m.Topic()), d, q.sendQueue)
			case DeviceClassParticulateSize:
				q.sensorMap[d.StateTopic] = NewParticulateSensorCount10(q.l.Named(m.Topic()), d, q.sendQueue)
			case "": // ignore unrelated messages
				sensorNotFound = true
			default:
				sensorNotFound = true
				q.l.Infof("[%s] unknown device class [%s]", m.Topic(), d.DeviceClass)
			}
			q.Unlock()
			if !sensorNotFound {
				q.l.Infof("adding %s sensor under %s", d.DeviceClass, d.StateTopic)
			}
		}
	})
	// this path need to be pretty exact to not catch the discovery path from above
	q.client.Subscribe("+/sensor/+/state", 0, func(c mqtt.Client, m mqtt.Message) {
		if strings.HasPrefix(m.Topic(), "homeassistant/") {
			return
		}
		if strings.Contains(m.Topic(), "ignoreme/") {
			return
		}
		q.RLock() // optimize that lock out
		if f, ok := q.sensorMap[m.Topic()]; ok {
			if q.cfg.Debug {
				q.l.Debugf("sensor %s: %s", m.Topic(), string(m.Payload()))
			}
			err := f.ProcessMessage(m)
			if err != nil {
				q.l.Warnf("could not process message %s: %s\n", m.Topic(), string(m.Payload()))
			}
		} else if q.cfg.Debug {
			q.l.Warnf("unhandled sensor: %s: %s", m.Topic(), string(m.Payload()))
		}
		q.RUnlock()
	})
}
