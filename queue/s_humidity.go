package queue

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var DeviceClassHumidity DeviceClass = "humidity"

type HumiditySensor struct {
	device     string
	sensor     string
	conversion func(float64) float64
	queue      chan Metric
}

func (t *HumiditySensor) ProcessMessage(msg mqtt.Message) error {
	metric := Metric{
		Name: "humidity",
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
func NewHumiditySensor(log *zap.SugaredLogger, discovery ESPHomeDiscovery, out chan Metric) *HumiditySensor {
	ts := &HumiditySensor{device: discovery.Dev.Name, sensor: discovery.Name}
	ts.queue = out
	return ts
}
