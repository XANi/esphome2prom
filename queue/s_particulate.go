package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var DeviceClassParticulate25 DeviceClass = "pm25"
var DeviceClassParticulate1 DeviceClass = "pm1"
var DeviceClassParticulate4 DeviceClass = "pm4"
var DeviceClassParticulate10 DeviceClass = "pm10"
var DeviceClassParticulateSize DeviceClass = "aqi"

type ParticulateSensor struct {
	device string
	sensor string
	unit   string
	size   string
	queue  chan Metric
}

func (t *ParticulateSensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "air_quality",
		Labels: map[string]string{
			"device": t.device,
			"sensor": t.sensor,
			"unit":   t.unit,
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
func NewParticulateSensor1(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *ParticulateSensor {
	ts := &ParticulateSensor{
		device: discovery.Dev.Name,
		sensor: discovery.Name,
		unit:   discovery.Unit,
	}
	ts.queue = out
	return ts
}

func NewParticulateSensor25(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *ParticulateSensor {
	ts := &ParticulateSensor{
		device: discovery.Dev.Name,
		sensor: discovery.Name,
		unit:   discovery.Unit,
	}
	ts.queue = out
	return ts
}
func NewParticulateSensor4(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *ParticulateSensor {
	ts := &ParticulateSensor{
		device: discovery.Dev.Name,
		sensor: discovery.Name,
		unit:   discovery.Unit,
	}
	ts.queue = out
	return ts
}
func NewParticulateSensor10(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *ParticulateSensor {
	ts := &ParticulateSensor{
		device: discovery.Dev.Name,
		sensor: discovery.Name,
		unit:   discovery.Unit,
	}
	ts.queue = out
	return ts
}

func NewParticulateSensorCount10(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *ParticulateSensor {
	ts := &ParticulateSensor{
		device: discovery.Dev.Name,
		sensor: discovery.Name,
		unit:   discovery.Unit,
	}
	ts.queue = out
	return ts
}
