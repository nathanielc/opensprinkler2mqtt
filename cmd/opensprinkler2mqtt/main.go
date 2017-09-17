package main

import (
	"log"
	"os"

	"github.com/nathanielc/opensprinkler2mqtt"
)

func main() {
	c := opensprinkler2mqtt.Config{
		OpenSprinklerURL: os.Getenv("OPEN_SPRINKLER_URL"),
		MQTTURL:          os.Getenv("MQTT_URL"),
		MQTTPrefix:       os.Getenv("MQTT_PREFIX"),
	}
	c.ApplyDefaults()
	s := opensprinkler2mqtt.New(c)
	log.Println("Running...")
	if err := s.Run(); err != nil {
		log.Fatal(err)
	}
}
