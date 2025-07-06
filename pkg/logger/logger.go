package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger interface {
	Debug() *zerolog.Event
	Info() *zerolog.Event
	Warn() *zerolog.Event
	Error() *zerolog.Event
	Fatal() *zerolog.Event
	With() zerolog.Context
	WithContext(ctx context.Context) Logger
}

type logger struct {
	zl zerolog.Logger
}

func New(level string) Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano

	logLevel, err := zerolog.ParseLevel(level)
	if err != nil {
		logLevel = zerolog.InfoLevel
	}

	zl := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	return &logger{zl: zl}
}

func (l *logger) Debug() *zerolog.Event {
	return l.zl.Debug()
}

func (l *logger) Info() *zerolog.Event {
	return l.zl.Info()
}

func (l *logger) Warn() *zerolog.Event {
	return l.zl.Warn()
}

func (l *logger) Error() *zerolog.Event {
	return l.zl.Error()
}

func (l *logger) Fatal() *zerolog.Event {
	return l.zl.Fatal()
}

func (l *logger) With() zerolog.Context {
	return l.zl.With()
}

func (l *logger) WithContext(ctx context.Context) Logger {
	return &logger{zl: l.zl.With().Ctx(ctx).Logger()}
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return log.With().Str("request_id", requestID).Logger().WithContext(ctx)
}

func FromContext(ctx context.Context) Logger {
	return &logger{zl: *log.Ctx(ctx)}
}
