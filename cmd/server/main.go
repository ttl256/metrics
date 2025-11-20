package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/ttl256/metrics/internal/config"
	"github.com/ttl256/metrics/internal/handler"
	"github.com/ttl256/metrics/internal/logger"
	"github.com/ttl256/metrics/internal/repository"
	"github.com/ttl256/metrics/internal/service"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", err)
		slog.Default().Error("", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	err := logger.Initialize("debug")
	if err != nil {
		return fmt.Errorf("setting logger: %w", err)
	}
	log := slog.Default()
	cfg := config.DefaultServer()
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	err = cfg.ApplyFlags(fs, os.Args[1:])
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

	var repo service.MetricsRepository
	if cfg.DB.DSN != "" { //nolint: nestif //let me be
		opts := repository.DBOptions{
			DSN:                   cfg.DB.DSN,
			ApplicationName:       cfg.DB.ApplicationName,
			ConnectTimeout:        cfg.DB.ConnectTimeout,
			StatementTimeout:      cfg.DB.StatementTimeout,
			LockTimeout:           cfg.DB.LockTimeout,
			IdleInTxTimeout:       cfg.DB.IdleInTxSessionTimeout,
			PoolMaxConns:          cfg.DB.Pool.MaxConns,
			PoolMinConns:          cfg.DB.Pool.MinConns,
			MaxConnLifetime:       cfg.DB.Pool.MaxConnLifetime,
			MaxConnLifetimeJitter: cfg.DB.Pool.MaxConnLifetimeJitter,
			MaxConnIdleTime:       cfg.DB.Pool.MaxConnIdleTime,
			HealthCheckPeriod:     cfg.DB.Pool.HealthCheckPeriod,
		}
		var repoDB *repository.DBStorage
		repoDB, err = repository.NewDBStorage(ctx, opts)
		if err != nil {
			return fmt.Errorf("opening db: %w", err)
		}
		defer repoDB.Close()

		const repoPingTimeout = 30 * time.Second
		pingCtx, cancel := context.WithTimeout(ctx, repoPingTimeout)
		defer cancel()
		err = repoDB.RepoPing(pingCtx)
		if err != nil {
			return fmt.Errorf("pinging repo: %w", err)
		}
		err = repoDB.Migrate()
		if err != nil {
			return fmt.Errorf("migration: %w", err)
		}
		repo = repoDB
	} else {
		repo, err = repository.NewFileStorage(cfg.FileStoragePath, cfg.StoreInterval, cfg.Restore)
		if err != nil {
			return fmt.Errorf("initiating file storage: %w", err)
		}
	}

	svc := service.NewService(repo)
	h := handler.NewHTTPHandler(svc, []byte(cfg.Key))

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      h.Routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second, //nolint: mnd //fine
		WriteTimeout: 30 * time.Second, //nolint: mnd //fine
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	const shutdownDuration = 10 * time.Second
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Info("starting server", slog.String("address", cfg.Address))
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		log.Info("received shutdown signal")
		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, shutdownDuration)
		defer shutdownCancel()
		if err = srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutting down server: %w", err)
		}
		log.Info("server is shutdown")
		return nil
	})

	err = g.Wait()
	if err != nil {
		return fmt.Errorf("waiting for server to shutdown: %w", err)
	}
	log.Info("exiting")
	return nil
}
