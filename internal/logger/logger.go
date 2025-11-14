package logger

import (
	"log/slog"
	"os"

	xerrors "github.com/pkg/errors"
)

func Initialize(level string) error {
	lvl := slog.Level(0)
	err := lvl.UnmarshalText([]byte(level))
	if err != nil {
		return xerrors.WithStack(err)
	}
	lVar := slog.LevelVar{}
	lVar.Set(lvl)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     &lVar,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				a.Value = slog.TimeValue(a.Value.Time().UTC())
			}
			return a
		},
	}))
	slog.SetDefault(logger)
	return nil
}
