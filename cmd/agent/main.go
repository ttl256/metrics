package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ttl256/metrics/internal/agent"
	"github.com/ttl256/metrics/internal/config"
	"github.com/ttl256/metrics/internal/logger"
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
	err = logger.Initialize(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("initiating logger: %w", err)
	}
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	a := agent.NewAgent(cfg.Endpoint, cfg.PollInterval, cfg.ReportInterval, cfg.RateLimit, []byte(cfg.Key))
	return fmt.Errorf("agent: %w", a.Run(ctx))
}
