package opensprinkler2mqtt

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/eclipse/paho.mqtt.golang"
	"github.com/nathanielc/smarthome"
)

type Server struct {
	c          Config
	home       smarthome.Server
	state      smarthome.ConnectionState
	lastStatus StationStatus

	stationsPath     string
	startStationPath string

	mu sync.Mutex
}

const (
	on  = "on"
	off = "off"
)

func New(c Config) *Server {
	mqtt.ERROR = log.New(os.Stderr, "[mqtt] E ", 0)
	opts := smarthome.DefaultMQTTClientOptions().
		AddBroker(c.MQTTURL).
		SetClientID("opensprinkler2mqtt")
	s := &Server{
		c:                c,
		state:            smarthome.Disconnected,
		stationsPath:     c.OpenSprinklerURL + "/js",
		startStationPath: c.OpenSprinklerURL + "/cm",
	}
	s.home = smarthome.NewServer(c.MQTTPrefix, s, opts)
	return s
}

func (s *Server) Run() error {
	if err := s.home.Connect(); err != nil {
		return err
	}

	// Tick once on startup
	newStatus, err := s.tick(time.Now())
	if err != nil {
		log.Println(err)
		s.updateConnectionState(smarthome.Disconnected)
	} else {
		s.updateConnectionState(smarthome.Connected)
		s.lastStatus = newStatus
	}

	ticker := time.NewTicker(s.c.PollInterval)
	for t := range ticker.C {
		newStatus, err := s.tick(t)
		if err != nil {
			log.Println(err)
			s.updateConnectionState(smarthome.Disconnected)
			continue
		}
		s.updateConnectionState(smarthome.Connected)
		s.lastStatus = newStatus
	}
	return nil
}

func (s *Server) tick(t time.Time) (StationStatus, error) {
	r, err := http.Get(s.stationsPath)
	if err != nil {
		return StationStatus{}, err
	}
	defer r.Body.Close()
	newStatus := StationStatus{}
	if err := json.NewDecoder(r.Body).Decode(&newStatus); err != nil {
		return StationStatus{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.lastStatus.Stations) != len(newStatus.Stations) {
		n := make([]int, len(newStatus.Stations))
		copy(n, s.lastStatus.Stations)
		s.lastStatus.Stations = n
	}
	for i := range newStatus.Stations {
		if s.lastStatus.Stations[i] != newStatus.Stations[i] {
			v := on
			if newStatus.Stations[i] == 0 {
				v = off
			}
			s.home.PublishStatus(
				strconv.Itoa(i),
				smarthome.Value{
					Value: v,
					Time:  t,
				},
			)
		}
	}
	return newStatus, nil
}

func (s *Server) updateConnectionState(newstate smarthome.ConnectionState) {
	if newstate != s.state {
		s.state = newstate
		s.home.PublishHWStatus(s.state)
	}
}

type StationStatus struct {
	Stations []int `json:"sn"`
}

func (s *Server) Set(toplevel string, item string, value interface{}) {
	i, err := strconv.Atoi(item)
	if err != nil {
		return
	}
	vStr, ok := value.(string)
	if !ok {
		return
	}
	var v int
	var enabled, duration string
	switch vStr {
	case off:
		v = 0
		enabled = "0"
	case on:
		v = 1
		enabled = "1"
		duration = "60"
	default:
		v = 1
		enabled = "1"
		duration = vStr
	}

	params := url.Values{}
	params.Set("sid", item)
	params.Set("en", enabled)
	params.Set("t", duration)

	u := s.startStationPath + "?" + params.Encode()
	r, err := http.Get(u)
	if err != nil {
		log.Println(err)
		return
	}
	defer r.Body.Close()

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastStatus.Stations[i] = v

	s.home.PublishStatus(
		item,
		smarthome.Value{
			Value: value,
			Time:  time.Now(),
		},
	)
}

func (s *Server) Get(toplevel string, item string) (smarthome.Value, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	i, err := strconv.Atoi(item)
	if err != nil {
		return smarthome.Value{}, false
	}

	v := on
	if s.lastStatus.Stations[i] == 0 {
		v = off
	}
	return smarthome.Value{
		Value: v,
		Time:  time.Now(),
	}, true
}

func (s *Server) Command(toplevel string, cmd []byte) {
}
