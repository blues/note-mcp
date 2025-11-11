package lib

import (
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger initializes the global logger with the specified log level
// Valid levels: trace, debug, info, warn, error, fatal, panic
func InitLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	var output io.Writer = os.Stderr

	// Parse log level from string
	logLevel := parseLogLevel(level)

	// Enable pretty printing for debug and trace levels
	if logLevel <= zerolog.TraceLevel {
		output = zerolog.ConsoleWriter{Out: os.Stderr}
	}

	// Set the global logger
	log.Logger = zerolog.New(output).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	log.Info().Str("set-level", level).Msg("Logger initialized")
}

// parseLogLevel converts a string to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
