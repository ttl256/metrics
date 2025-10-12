package main

import (
	"context"
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
	cfg, err := config.NewAgent()
	if err != nil {
		return fmt.Errorf("creating agent: %w", err)
	}
	ctx := context.Background()
	agent := agent.NewAgent(cfg)
	return fmt.Errorf("agent: %w", agent.Run(ctx))
}
