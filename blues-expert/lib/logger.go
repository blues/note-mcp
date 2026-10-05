package lib

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
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

// NewSlogLogger returns an slog.Logger that writes through the global zerolog
// logger, so log output from the MCP SDK shares the server's format and level.
// Records below minLevel are dropped even when zerolog would emit them.
// Attributes inside slog groups are flattened to dotted keys.
func NewSlogLogger(minLevel slog.Level) *slog.Logger {
	return slog.New(&zerologHandler{minLevel: minLevel})
}

type zerologHandler struct {
	minLevel slog.Level
	attrs    []prefixedAttr
	prefix   string
}

type prefixedAttr struct {
	prefix string
	attr   slog.Attr
}

func (h *zerologHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.minLevel && slogToZerologLevel(level) >= log.Logger.GetLevel()
}

func (h *zerologHandler) Handle(_ context.Context, r slog.Record) error {
	event := log.WithLevel(slogToZerologLevel(r.Level)).Str("component", "mcp-sdk")
	for _, pa := range h.attrs {
		event = addSlogAttr(event, pa.prefix, pa.attr)
	}
	r.Attrs(func(a slog.Attr) bool {
		event = addSlogAttr(event, h.prefix, a)
		return true
	})
	event.Msg(r.Message)
	return nil
}

func (h *zerologHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.attrs = slices.Clip(h.attrs)
	for _, a := range attrs {
		next.attrs = append(next.attrs, prefixedAttr{prefix: h.prefix, attr: a})
	}
	return &next
}

func (h *zerologHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	next := *h
	next.prefix = h.prefix + name + "."
	return &next
}

func addSlogAttr(event *zerolog.Event, prefix string, a slog.Attr) *zerolog.Event {
	v := a.Value.Resolve()
	if v.Kind() == slog.KindGroup {
		groupPrefix := prefix
		if a.Key != "" {
			groupPrefix += a.Key + "."
		}
		for _, ga := range v.Group() {
			event = addSlogAttr(event, groupPrefix, ga)
		}
		return event
	}
	if a.Key == "" {
		return event
	}
	key := prefix + a.Key
	switch key {
	case zerolog.LevelFieldName, zerolog.MessageFieldName, zerolog.TimestampFieldName, "component":
		key = "sdk_" + key
	}
	switch v.Kind() {
	case slog.KindString:
		return event.Str(key, v.String())
	case slog.KindInt64:
		return event.Int64(key, v.Int64())
	case slog.KindUint64:
		return event.Uint64(key, v.Uint64())
	case slog.KindFloat64:
		return event.Float64(key, v.Float64())
	case slog.KindBool:
		return event.Bool(key, v.Bool())
	case slog.KindDuration:
		return event.Dur(key, v.Duration())
	case slog.KindTime:
		return event.Time(key, v.Time())
	}
	if err, ok := v.Any().(error); ok {
		return event.AnErr(key, err)
	}
	return event.Interface(key, v.Any())
}

func slogToZerologLevel(level slog.Level) zerolog.Level {
	switch {
	case level >= slog.LevelError:
		return zerolog.ErrorLevel
	case level >= slog.LevelWarn:
		return zerolog.WarnLevel
	case level >= slog.LevelInfo:
		return zerolog.InfoLevel
	default:
		return zerolog.DebugLevel
	}
}
