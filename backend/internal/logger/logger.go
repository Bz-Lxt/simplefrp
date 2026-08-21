package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"simplefrp/internal/clock"
)

func New(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
				a.Value = slog.StringValue(a.Value.Time().In(clock.Beijing).Format("2006-01-02 15:04:05.000"))
			}
			return a
		},
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
