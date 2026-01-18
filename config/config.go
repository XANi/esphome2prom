package config

import (
	"github.com/goccy/go-yaml"
	"os"
)

type ConfigWithDefault interface {
	GetDefaultConfig() string
}

type Config struct {
	ListenAddress      string `yaml:"address"`
	MQTTAddress        string `yaml:"mqtt_address"`
	PrometheusWriteURL string `yaml:"prometheus_write_url"`
	PrometheusPrefix   string `yaml:"prometheus_prefix"`
	Debug              bool   `yaml:"debug"`
	PProfAddress       string `yaml:"pprof_address"`
	ExtraLabels        map[string]string
}

func (c *Config) GetDefaultConfig() string {
	h, _ := os.Hostname()
	cfg := Config{
		ListenAddress:      "127.0.0.1:3001",
		MQTTAddress:        "tcp://mqtt:mqtt@example.com:1880",
		PrometheusWriteURL: "http://cthulhu.home.zxz.li:8480/insert/999:0/prometheus/api/v1/write",
		PrometheusPrefix:   "example_",
		Debug:              true,
		PProfAddress:       "127.0.0.1:6060",
		ExtraLabels: map[string]string{
			"host": h,
		},
	}
	b, _ := yaml.Marshal(&cfg)
	return string(b)
}
