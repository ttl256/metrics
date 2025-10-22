package config_test

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ttl256/metrics/internal/config"
)

func TestServerConfig(t *testing.T) {
	t.Run("empty env", func(t *testing.T) {
		cfg := config.DefaultServer()
		cfg.ApplyEnv()
		want := config.DefaultServer()
		assert.Equal(t, want, cfg)
	})
	t.Run("set only env", func(t *testing.T) {
		t.Setenv("ADDRESS", "localhost:9191")
		cfg := config.DefaultServer()
		cfg.ApplyEnv()
		want := &config.Server{Address: "localhost:9191"}
		assert.Equal(t, want, cfg)
	})
	t.Run("set only flag", func(t *testing.T) {
		fs := flag.NewFlagSet("", flag.ContinueOnError)
		cfg := config.DefaultServer()
		cfg.ApplyFlags(fs, []string{"-a=localhost:9191"})
		want := &config.Server{Address: "localhost:9191"}
		assert.Equal(t, want, cfg)
	})
	t.Run("env var over flag", func(t *testing.T) {
		cfg := config.DefaultServer()

		fs := flag.NewFlagSet("", flag.ContinueOnError)
		cfg.ApplyFlags(fs, []string{"-a=localhost:9191"})

		t.Setenv("ADDRESS", "localhost:9192")
		cfg.ApplyEnv()

		want := &config.Server{Address: "localhost:9192"}
		assert.Equal(t, want, cfg)
	})
	t.Run("env var is missing but flag provided", func(t *testing.T) {
		cfg := config.DefaultServer()

		fs := flag.NewFlagSet("", flag.ContinueOnError)
		cfg.ApplyFlags(fs, []string{"-a=localhost:9191"})

		want := &config.Server{Address: "localhost:9191"}
		assert.Equal(t, want, cfg)
	})
}
