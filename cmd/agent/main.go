package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/ttl256/metrics/internal/agent"
	"github.com/ttl256/metrics/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.DefaultAgent()
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	err := cfg.ApplyFlags(fs, os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return fmt.Errorf("initiating app: %w", err)
	}
	if err = cfg.ApplyEnv(); err != nil {
		return fmt.Errorf("initiating app: %w", err)
	}
	ctx := context.Background()
	a := agent.NewAgent(cfg.Endpoint, cfg.PollInterval, cfg.ReportInterval)
	return fmt.Errorf("agent: %w", a.Run(ctx))
}
