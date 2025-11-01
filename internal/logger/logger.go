package logger

import (
	"time"

	xerrors "github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger = zap.NewNop() //nolint: gochecknoglobals //TODO

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return xerrors.WithStack(err)
	}
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.UTC().Format(time.RFC3339))
	}
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return xerrors.WithStack(err)
	}
	Log = zl
	return nil
}
