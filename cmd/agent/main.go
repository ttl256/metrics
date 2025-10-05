package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/ttl256/metrics/internal/agent"
)

const (
	defaultPollInterval   = 2 * time.Second
	defaultReportInterval = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	_url, _ := url.Parse("http://localhost:8080")
	agent := agent.NewAgent(*_url, defaultPollInterval, defaultReportInterval)
	return fmt.Errorf("agent: %w", agent.Run(ctx))
}
