package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"time"
)

var DeviceClassSignalStrength DeviceClass = "signal_strength"

type SignalStrengthSensor struct {
	device string
	sensor string
	unit   string
	queue  chan Metric
}

func (t *SignalStrengthSensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "signal_strength",
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
func NewSignalStrengthSensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *SignalStrengthSensor {
	s := &SignalStrengthSensor{device: discovery.Dev.Name, sensor: discovery.Name, unit: discovery.Unit}
	//normalize unit
	switch strings.ToLower(s.unit) {
	case "dbm":
		s.unit = "dBm"
	case "db":
		s.unit = "dB"
	}

	s.queue = out
	return s
}
