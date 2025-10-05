package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

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
	app := handler.NewApp(service.NewService(repository.NewMemStorage()))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", app.HealthHandler)
	mux.Handle("POST /update/{type}", http.NotFoundHandler())
	mux.HandleFunc("POST /update/{type}/{name}/{value}", app.UpdateHandler)
	mux.HandleFunc("GET /{id}", app.GetHandler)
	mux.HandleFunc("GET /all", app.GetAllHandler)

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("serving HTTP: %w", err)
	}
	return nil
}
