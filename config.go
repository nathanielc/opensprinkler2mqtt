package opensprinkler2mqtt

import "time"

type Config struct {
	OpenSprinklerURL string
	MQTTURL          string
	MQTTPrefix       string
	PollInterval     time.Duration
}

func (c *Config) ApplyDefaults() {
	if c.OpenSprinklerURL == "" {
		c.OpenSprinklerURL = "http://localhost:8080"
	}
	if c.MQTTURL == "" {
		c.MQTTURL = "tcp://localhost:1883"
	}
	if c.MQTTPrefix == "" {
		c.MQTTPrefix = "opensprinkler"
	}
	if c.PollInterval == 0 {
		c.PollInterval = 10 * time.Second
	}
}
