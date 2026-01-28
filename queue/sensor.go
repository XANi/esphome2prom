package queue

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
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
