package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type Agent struct {
	Endpoint       string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func NewAgent() (*Agent, error) {
	address := flag.String("a", "http://localhost:8080", "server address to listen on")
	reportIntervalFlag := flag.String("r", "10", "metric report interval")
	pollIntervalFlag := flag.String("p", "2", "metric poll interval")
	flag.Parse()
	if !strings.HasPrefix(*address, "http://") {
		*address = "http://" + *address
	}
	reportInterval, err := time.ParseDuration(*reportIntervalFlag + "s")
	if err != nil {
		return nil, fmt.Errorf("parsing report interval flag: %w", err)
	}
	pollInterval, err := time.ParseDuration(*pollIntervalFlag + "s")
	if err != nil {
		return nil, fmt.Errorf("parsing poll interval flag: %w", err)
	}
	return &Agent{
		Endpoint:       *address,
		ReportInterval: reportInterval,
		PollInterval:   pollInterval,
	}, nil
}
