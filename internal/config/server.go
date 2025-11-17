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
	DB              DBConfig
}

func DefaultServer() *Server {
	return &Server{
		Address:         "localhost:8080",
		StoreInterval:   300 * time.Second, //nolint: mnd //fine
		FileStoragePath: filepath.Join(os.TempDir(), "metrics_state.json"),
		Restore:         false,
		DB: DBConfig{
			DSN:                    "",
			ApplicationName:        "",
			ConnectTimeout:         0,
			StatementTimeout:       0,
			LockTimeout:            0,
			IdleInTxSessionTimeout: 0,
			Pool: DBPoolConfig{
				MaxConns:              0,
				MinConns:              0,
				MaxConnLifetime:       0,
				MaxConnLifetimeJitter: 0,
				MaxConnIdleTime:       0,
				HealthCheckPeriod:     0,
			},
		},
	}
}

func (a *Server) ApplyEnv() error { //nolint: gocognit,funlen //let me be
	if v, ok := os.LookupEnv("ADDRESS"); ok {
		a.Address = v
	}
	if v, ok := os.LookupEnv("STORE_INTERVAL"); ok {
		storeInterval, err := time.ParseDuration(v + "s")
		if err != nil {
			return fmt.Errorf("parsing STORE_INTERVAL: %w", err)
		}
		a.StoreInterval = storeInterval
	}
	if v, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		a.FileStoragePath = v
	}
	if v, ok := os.LookupEnv("RESTORE"); ok {
		restore, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("parsing RESTORE: %w", err)
		}
		a.Restore = restore
	}
	if v, ok := os.LookupEnv("DATABASE_DSN"); ok {
		a.DB.DSN = v
	}
	if v, ok := os.LookupEnv("DB_APP_NAME"); ok {
		a.DB.ApplicationName = v
	}
	if v, ok := os.LookupEnv("DB_CONNECT_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_CONNECT_TIMEOUT: %w", err)
		}
		a.DB.ConnectTimeout = d
	}
	if v, ok := os.LookupEnv("DB_STATEMENT_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_STATEMENT_TIMEOUT: %w", err)
		}
		a.DB.StatementTimeout = d
	}
	if v, ok := os.LookupEnv("DB_LOCK_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_LOCK_TIMEOUT: %w", err)
		}
		a.DB.LockTimeout = d
	}
	if v, ok := os.LookupEnv("DB_IDLE_IN_TX_TIMEOUT"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_IDLE_IN_TX_TIMEOUT: %w", err)
		}
		a.DB.IdleInTxSessionTimeout = d
	}
	if v, ok := os.LookupEnv("DB_POOL_MAX_CONNS"); ok {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_MAX_CONNS: %w", err)
		}
		a.DB.Pool.MaxConns = int32(i)
	}
	if v, ok := os.LookupEnv("DB_POOL_MIN_CONNS"); ok {
		i, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_MIN_CONNS: %w", err)
		}
		a.DB.Pool.MinConns = int32(i)
	}
	if v, ok := os.LookupEnv("DB_POOL_MAX_CONN_LIFETIME"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_MAX_CONN_LIFETIME: %w", err)
		}
		a.DB.Pool.MaxConnLifetime = d
	}
	if v, ok := os.LookupEnv("DB_POOL_MAX_CONN_LIFETIME_JITTER"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_MAX_CONN_LIFETIME_JITTER: %w", err)
		}
		a.DB.Pool.MaxConnLifetimeJitter = d
	}
	if v, ok := os.LookupEnv("DB_POOL_MAX_CONN_IDLE_TIME"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_MAX_CONN_IDLE_TIME: %w", err)
		}
		a.DB.Pool.MaxConnIdleTime = d
	}
	if v, ok := os.LookupEnv("DB_POOL_HEALTH_CHECK_PERIOD"); ok {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("parsing DB_POOL_HEALTH_CHECK_PERIOD: %w", err)
		}
		a.DB.Pool.HealthCheckPeriod = d
	}
	return nil
}

func (a *Server) ApplyFlags(fs *flag.FlagSet, args []string) error {
	addr := fs.String("a", a.Address, "server address to listen on")
	storeIntervalFlag := fs.Duration("i", a.StoreInterval, "store interval")
	fileStoragePathFlag := fs.String("f", a.FileStoragePath, "file to store metrics")
	restoreFlag := fs.Bool("r", a.Restore, "read metrics from a file on startup")
	dsnFlag := fs.String("d", a.DB.DSN, "database DSN")
	dbAppNameFlag := fs.String("db-app-name", a.DB.ApplicationName, "db application_name")
	dbConnectTimeoutFlag := fs.Duration("db-connect-timeout", a.DB.ConnectTimeout, "db connect_timeout")
	dbStatementTimeoutFlag := fs.Duration("db-statement-timeout", a.DB.StatementTimeout, "db statement_timeout")
	dbLockTimeoutFlag := fs.Duration("db-lock-timeout", a.DB.LockTimeout, "db lock_timeout")
	dbIdleInTxTimeoutFlag := fs.Duration(
		"db-idle-in-tx-timeout", a.DB.IdleInTxSessionTimeout, "db idle_in_transaction_session_timeout",
	)
	poolMaxConnsFlag := fs.Int("db-pool-max-conns", int(a.DB.Pool.MaxConns), "pool max connections")
	poolMinConnsFlag := fs.Int("db-pool-min-conns", int(a.DB.Pool.MinConns), "pool min connections")
	poolMaxConnLifetimeFlag := fs.Duration(
		"db-pool-max-conn-lifetime", a.DB.Pool.MaxConnLifetime, "pool max conn lifetime",
	)
	poolMaxConnLifetimeJitterFlag := fs.Duration(
		"db-pool-max-conn-lifetime-jitter", a.DB.Pool.MaxConnLifetimeJitter, "pool max conn lifetime jitter",
	)
	poolMaxConnIdleTimeFlag := fs.Duration(
		"db-pool-max-conn-idle-time", a.DB.Pool.MaxConnIdleTime, "pool max conn idle time",
	)
	poolHealthCheckPeriodFlag := fs.Duration(
		"db-pool-health-check-period", a.DB.Pool.HealthCheckPeriod, "pool health check period",
	)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing command line flags: %w", err)
	}
	a.Address = *addr
	a.StoreInterval = *storeIntervalFlag
	a.FileStoragePath = *fileStoragePathFlag
	a.Restore = *restoreFlag
	a.DB.DSN = *dsnFlag
	a.DB.ApplicationName = *dbAppNameFlag
	a.DB.ConnectTimeout = *dbConnectTimeoutFlag
	a.DB.StatementTimeout = *dbStatementTimeoutFlag
	a.DB.LockTimeout = *dbLockTimeoutFlag
	a.DB.IdleInTxSessionTimeout = *dbIdleInTxTimeoutFlag
	a.DB.Pool.MaxConns = int32(*poolMaxConnsFlag) //nolint: gosec //fine
	a.DB.Pool.MinConns = int32(*poolMinConnsFlag) //nolint: gosec //fine)
	a.DB.Pool.MaxConnLifetime = *poolMaxConnLifetimeFlag
	a.DB.Pool.MaxConnLifetimeJitter = *poolMaxConnLifetimeJitterFlag
	a.DB.Pool.MaxConnIdleTime = *poolMaxConnIdleTimeFlag
	a.DB.Pool.HealthCheckPeriod = *poolHealthCheckPeriodFlag
	return nil
}

type DBPoolConfig struct {
	MaxConns              int32
	MinConns              int32
	MaxConnLifetime       time.Duration
	MaxConnLifetimeJitter time.Duration
	MaxConnIdleTime       time.Duration
	HealthCheckPeriod     time.Duration
}

type DBConfig struct {
	DSN                    string
	ApplicationName        string
	ConnectTimeout         time.Duration
	StatementTimeout       time.Duration
	LockTimeout            time.Duration
	IdleInTxSessionTimeout time.Duration
	Pool                   DBPoolConfig
}
