package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Server struct {
	Address         string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DSN             string
}

func DefaultServer() *Server {
	return &Server{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second, //nolint: mnd //fine
		FileStoragePath: filepath.Join(os.TempDir(), "metrics_state.json"),
		Restore:         false,
		DSN:             "",
	}
}

func (a *Server) ApplyEnv() error {
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		a.Address = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		storeInterval, err := time.ParseDuration(v + "s")
		if err != nil {
			return fmt.Errorf("parsing store interval: %w", err)
		}
		a.StoreInterval = storeInterval
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		a.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		restore, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("parsing restore: %w", err)
		}
		a.Restore = restore
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		a.DSN = v
	}
	return nil
}

func (a *Server) ApplyFlags(fs *flag.FlagSet, args []string) error {
	addr := fs.String("a", a.Address, "server address to listen on")
	storeIntervalFlag := fs.Duration("i", a.StoreInterval, "store interval")
	fileStoragePathFlag := fs.String("f", a.FileStoragePath, "file to store metrics")
	restoreFlag := fs.Bool("r", a.Restore, "read metrics from a file on startup")
	dsnFlag := fs.String("d", a.DSN, "database DSN")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing command line flags: %w", err)
	}
	a.Address = *addr
	a.StoreInterval = *storeIntervalFlag
	a.FileStoragePath = *fileStoragePathFlag
	a.Restore = *restoreFlag
	a.DSN = *dsnFlag

	return nil
}
