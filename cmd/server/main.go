package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

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
	cfg := config.DefaultServer()
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

	repo := repository.NewMemStorage()
	svc := service.NewService(repo)
	h := handler.NewHTTPHandler(svc)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      h.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}
