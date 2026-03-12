package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Setup initializes and sets the default slog logger.
// level: "debug", "info", "warn", "error"
// format: "json" or "text"
// logFile: path to append log file (empty = stdout only)
func Setup(level, format, logFile string) error {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return fmt.Errorf("unknown log level: %q", level)
	}

	var w io.Writer = os.Stdout
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		w = io.MultiWriter(os.Stdout, f)
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	switch format {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		return fmt.Errorf("unknown log format: %q", format)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}
