package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var DeviceClassVoltage DeviceClass = "voltage"

type VoltageSensor struct {
	device     string
	sensor     string
	conversion func(float64) float64
	queue      chan Metric
}

func (t *VoltageSensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "voltage",
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
func NewVoltageSensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *VoltageSensor {
	s := &VoltageSensor{device: discovery.Dev.Name, sensor: discovery.Name}
	switch strings.ToLower(discovery.Unit) {
	case "v":
		s.conversion = func(v float64) float64 { return v }
	case "mv":
		s.conversion = func(v float64) float64 { return v / 1000 }
	case "uv", "μv":
		s.conversion = func(v float64) float64 { return v / (1000 * 1000) }
	case "kv":
		s.conversion = func(v float64) float64 { return v * 1000 }
	}
	s.queue = out
	return s
}
