package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"math"
	"strconv"
	"strings"
	"time"
)

type Sensor interface {
	ProcessMessage(msg mqtt.Message) error
}

type Metric struct {
	Name   string
	Labels map[string]string
	Value  float64
	TS     time.Time
}

type DeviceClass string

var DeviceClassTemperature DeviceClass = "temperature"

type TemperatureSensor struct {
	device     string
	sensor     string
	conversion func(float64) float64
	queue      chan Metric
}

func (t *TemperatureSensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "temperature",
		Labels: map[string]string{
			"device": t.device,
			"sensor": t.sensor,
		},
	}
	v, err := strconv.ParseFloat(string(msg.Payload()), 64)
	if err != nil {
		return fmt.Errorf("error parsing[%s]:%s", string(msg.Payload()), err)
	}
	metric.Value = t.conversion(v)
	metric.TS = time.Now()
	select {
	case t.queue <- metric:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("timeout on send queue")
	}
}
func NewTemperatureSensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *TemperatureSensor {
	ts := &TemperatureSensor{device: discovery.Dev.Name, sensor: discovery.Name}
	if strings.Contains(strings.ToUpper(discovery.Unit), "K") {
		ts.conversion = func(k float64) (c float64) { return k - 273.15 }
	} else if strings.Contains(strings.ToUpper(discovery.Unit), "F") {
		ts.conversion = func(f float64) (c float64) {
			return float64(math.Round((f-32.0)*(5.0/9.0)*10.0)) / 10.0
		}
	} else {
		ts.conversion = func(c float64) float64 { return c }
	}
	ts.queue = out
	return ts
}
