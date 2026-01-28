package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var DeviceClassPressure DeviceClass = "pressure"

type PressureSensor struct {
	device     string
	sensor     string
	conversion func(float64) float64
	queue      chan Metric
}

func (t *PressureSensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "pressure",
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
func NewPressureSensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *PressureSensor {
	ts := &PressureSensor{device: discovery.Dev.Name, sensor: discovery.Name}
	if !strings.Contains(strings.ToLower(discovery.Unit), "hpa") {
		log.Warnf("sensor [%s] does not contain hPa unit, add conversion", discovery)
	}
	ts.conversion = func(c float64) float64 { return c }
	ts.queue = out
	return ts
}
