package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

type Agent struct {
	Endpoint       string
	ReportInterval time.Duration
	PollInterval   time.Duration
	Key            string
}

func DefaultAgent() *Agent {
	return &Agent{
		Endpoint:       "http://localhost:8080",
		ReportInterval: 10 * time.Second, //nolint: mnd //default value
		PollInterval:   2 * time.Second,  //nolint: mnd //default value
		Key:            "",
	}
}

func (a *Agent) ApplyEnv() error {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		if !strings.HasPrefix(v, "http://") {
			v = "http://" + v
		}
		a.Endpoint = v
	}
	if v, ok := os.LookupEnv("REPORT_INTERVAL"); ok {
		reportInterval, err := time.ParseDuration(v + "s")
		if err != nil {
			return fmt.Errorf("parsing report interval: %w", err)
		}
		a.ReportInterval = reportInterval
	}
	if v, ok := os.LookupEnv("POLL_INTERVAL"); ok {
		pollInterval, err := time.ParseDuration(v + "s")
		if err != nil {
			return fmt.Errorf("parsing poll interval: %w", err)
		}
		a.PollInterval = pollInterval
	}
	if v, ok := os.LookupEnv("KEY"); ok {
		a.Key = v
	}
	return nil
}

func (a *Agent) ApplyFlags(fs *flag.FlagSet, args []string) error {
	address := fs.String("a", a.Endpoint, "server address to listen on")
	reportIntervalFlag := fs.String("r", a.ReportInterval.String(), "metric report interval")
	pollIntervalFlag := fs.String("p", a.PollInterval.String(), "metric poll interval")
	keyFlag := fs.String("k", a.Key, "key to hash metrics")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing command line flags: %w", err)
	}
	if !strings.HasPrefix(*address, "http://") {
		*address = "http://" + *address
	}
	a.Endpoint = *address

	if !strings.HasSuffix(*reportIntervalFlag, "s") {
		*reportIntervalFlag += "s"
	}
	reportInterval, err := time.ParseDuration(*reportIntervalFlag)
	if err != nil {
		return fmt.Errorf("parsing report interval flag: %w", err)
	}
	a.ReportInterval = reportInterval

	if !strings.HasSuffix(*pollIntervalFlag, "s") {
		*pollIntervalFlag += "s"
	}
	pollInterval, err := time.ParseDuration(*pollIntervalFlag)
	if err != nil {
		return fmt.Errorf("parsing poll interval flag: %w", err)
	}
	a.PollInterval = pollInterval

	a.Key = *keyFlag
	return nil
}
