package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/sury-dev/portfolio-server/internal/config"
)

// NewLogger builds a zerolog logger from already-validated logging configuration
// and a service identity (SERVER.NAME) attached to every event.
//
// Construction is three independent steps so each concern stays small:
//
//  1. parseLogLevel  — config string  → zerolog.Level
//  2. createWriter   — destination    → io.Writer (and a closer if a file was opened)
//  3. createOutput   — format + writer → the stream zerolog actually writes to
//
// The returned closer must be called on shutdown. For stdout/stderr it is a
// no-op; for a file it closes the handle. If a later step fails after a file
// was opened, NewLogger closes it before returning the error so the caller
// does not have to.
func NewLogger(cfg config.LoggingConfig, service string) (*zerolog.Logger, func() error, error) {
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid log level: %w", err)
	}

	writer, closer, err := createWriter(cfg.Output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create writer: %w", err)
	}

	output, err := createOutput(cfg.Format, writer)
	if err != nil {
		return nil, closer, fmt.Errorf("failed to create output: %w", err)
	}

	// Timestamp and service identity are bound on the logger context so
	// every event carries them without callers repeating the fields.
	// service is SERVER.NAME: which process produced the line.
	logger := zerolog.New(output).Level(level).With().
		Timestamp().
		Str("service", service).
		Logger()
	return &logger, closer, nil
}

// parseLogLevel maps the config vocabulary onto zerolog's levels.
// "warning" is the name we store after config canonicalization; zerolog's
// matching level is WarnLevel.
func parseLogLevel(level string) (zerolog.Level, error) {
	switch level {
	case "debug":
		return zerolog.DebugLevel, nil
	case "info":
		return zerolog.InfoLevel, nil
	case "warning":
		return zerolog.WarnLevel, nil
	case "error":
		return zerolog.ErrorLevel, nil
	default:
		return zerolog.NoLevel, fmt.Errorf("unsupported log level %q", level)
	}
}

// createWriter opens the destination bytes will be written to.
//
// stdout and stderr are process-owned streams, so the closer is a no-op.
// Any other value is treated as a filesystem path (the same contract as
// LoggingConfig.validate). The file is created if missing and always
// appended to, so restarts do not wipe previous logs.
func createWriter(output string) (io.Writer, func() error, error) {
	switch output {
	case "stdout":
		return os.Stdout, func() error { return nil }, nil
	case "stderr":
		return os.Stderr, func() error { return nil }, nil
	default:
		file, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file %q: %w", output, err)
		}
		return file, file.Close, nil
	}
}

// createOutput wraps the destination writer according to FORMAT.
//
// json: zerolog's default encoder already emits one JSON object per line,
// so the raw writer is used as-is.
//
// text: ConsoleWriter sits in front of the raw writer and turns those JSON
// objects into a human-readable line. It always implements io.Writer, which
// is why this function returns io.Writer rather than ConsoleWriter — json
// cannot produce a ConsoleWriter, and a struct cannot be nil on error.
func createOutput(format string, writer io.Writer) (io.Writer, error) {
	switch format {
	case "json":
		return writer, nil
	case "text":
		return zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported log format %q", format)
	}
}
