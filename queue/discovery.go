package queue

type ESPHomeDiscovery struct {
	// https://www.home-assistant.io/integrations/sensor/#device-class
	DeviceClass DeviceClass `json:"dev_cla"`
	Unit        string      `json:"unit_of_meas"`
	// https://developers.home-assistant.io/docs/core/entity/sensor/#available-state-classes
	StateClass        string      `json:"stat_cla"`
	Name              string      `json:"name"`
	StateTopic        string      `json:"stat_t"`
	CommandTopic      string      `json:"cmd_t"`
	AvailabilityTopic string      `json:"avty_t"`
	UniqID            string      `json:"uniq_id"`
	Dev               *ESPHomeDev `json:"dev"`
}

type ESPHomeDev struct {
	ID              string     `json:"ids"`
	Name            string     `json:"name"`
	SoftwareVersion string     `json:"sw"`
	Model           string     `json:"mdl"`
	Manufacturer    string     `json:"mf"`
	Cns             [][]string `json:"cns"`
}
