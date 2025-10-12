package main

import (
	"fmt"
	"os"

	"github.com/ttl256/metrics/internal/config"
	"github.com/ttl256/metrics/internal/handler"
	"github.com/ttl256/metrics/internal/repository"
	"github.com/ttl256/metrics/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.NewApplication()
	app := handler.NewApp(cfg, service.NewService(repository.NewMemStorage()))

	return fmt.Errorf("app: %w", app.Run())
}
