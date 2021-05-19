package pgxutil

import (
	"context"

	log "github.com/gsmcwhirter/go-util/v7/logging"
	"github.com/gsmcwhirter/go-util/v7/logging/level"
	"github.com/jackc/pgx/v4"
)

type Logger struct {
	log.Logger
}

var _ pgx.Logger = (*Logger)(nil)

func (l *Logger) Log(ctx context.Context, lvl pgx.LogLevel, msg string, data map[string]interface{}) {
	lf := levelFunc(lvl)

	ld := make([]interface{}, 0, 2*len(data)+2)
	ld = append(ld, "pgx_level", lvl.String())

	for k, v := range data {
		ld = append(ld, k, v)
	}

	lf(l.Logger).Message(msg, ld...)
}

func levelFunc(lvl pgx.LogLevel) func(log.Logger) log.Logger {
	switch lvl {
	case pgx.LogLevelTrace:
		return level.Debug
	case pgx.LogLevelDebug:
		return level.Debug
	case pgx.LogLevelInfo:
		return level.Info
	case pgx.LogLevelWarn:
		return level.Info
	case pgx.LogLevelError:
		return level.Error
	default: // covers log level None also
		return func(l log.Logger) log.Logger { return l }
	}
}
