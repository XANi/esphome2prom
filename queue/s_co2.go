package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var DeviceClassCO2 DeviceClass = "carbon_dioxide"

type CO2Sensor struct {
	device     string
	sensor     string
	conversion func(float64) float64
	queue      chan Metric
}

func (t *CO2Sensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "co2",
		Labels: map[string]string{
			"device": t.device,
			"sensor": t.sensor,
		},
	}
	v, err := strconv.ParseFloat(string(msg.Payload()), 64)
	if err != nil {
		return fmt.Errorf("error parsing[%s]:%s", string(msg.Payload()), err)
	}
	metric.Value = v
	metric.TS = time.Now()
	select {
	case t.queue <- metric:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("timeout on send queue")
	}
}
func NewCO2Sensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *CO2Sensor {
	ts := &CO2Sensor{device: discovery.Dev.Name, sensor: discovery.Name}
	ts.queue = out
	return ts
}
