package bundle

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/open-policy-agent/opa/v1/logging"
)

// slogAdapter routes OPA's own logging through NACP's slog pipeline.
//
// This is not cosmetic. sdk.Options defaults Logger to a buffered logger which
// OPA discards once its plugins have started, so an instance built without an
// explicit logger silently swallows every bundle download failure, activation
// error and signature verification failure.
type slogAdapter struct {
	logger *slog.Logger
	fields []any
}

var _ logging.Logger = (*slogAdapter)(nil)

func newSlogAdapter(logger *slog.Logger) *slogAdapter {
	return &slogAdapter{logger: logger}
}

func (a *slogAdapter) Debug(format string, args ...any) { a.log(slog.LevelDebug, format, args...) }
func (a *slogAdapter) Info(format string, args ...any)  { a.log(slog.LevelInfo, format, args...) }
func (a *slogAdapter) Warn(format string, args ...any)  { a.log(slog.LevelWarn, format, args...) }
func (a *slogAdapter) Error(format string, args ...any) { a.log(slog.LevelError, format, args...) }

func (a *slogAdapter) log(level slog.Level, format string, args ...any) {
	if !a.logger.Enabled(context.Background(), level) {
		return
	}
	message := format
	if len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}
	a.logger.Log(context.Background(), level, message, a.fields...)
}

func (a *slogAdapter) WithFields(fields map[string]any) logging.Logger {
	merged := make([]any, 0, len(a.fields)+2*len(fields))
	merged = append(merged, a.fields...)
	for key, value := range fields {
		merged = append(merged, slog.Any(key, value))
	}
	return &slogAdapter{logger: a.logger, fields: merged}
}

// GetLevel reports the level OPA should assume. OPA uses it to decide whether
// to emit expensive debug payloads and whether to enable rego print statements,
// so it must reflect the slog handler rather than a fixed value.
func (a *slogAdapter) GetLevel() logging.Level {
	switch {
	case a.logger.Enabled(context.Background(), slog.LevelDebug):
		return logging.Debug
	case a.logger.Enabled(context.Background(), slog.LevelInfo):
		return logging.Info
	case a.logger.Enabled(context.Background(), slog.LevelWarn):
		return logging.Warn
	default:
		return logging.Error
	}
}

// SetLevel is a no-op: the level is owned by the slog handler, which NACP
// configures from telemetry.logging.level.
func (a *slogAdapter) SetLevel(logging.Level) {}
