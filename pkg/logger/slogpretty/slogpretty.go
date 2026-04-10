package slogpretty

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdLog "log"
	"log/slog"
	"os"
	"strings"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

const (
	colorReset  = "\033[0m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// PrettyHandlerOptions содержит опции для настройки PrettyHandler.
type PrettyHandlerOptions struct {
	SlogOpts *slog.HandlerOptions
}

// PrettyHandler форматирует лог-записи в цветном, читаемом виде.
type PrettyHandler struct {
	opts   PrettyHandlerOptions
	groups []string
	slog.Handler
	l     *stdLog.Logger
	attrs []slog.Attr
}

// NewPrettyHandler создаёт новый PrettyHandler.
func (opts PrettyHandlerOptions) NewPrettyHandler(out io.Writer) *PrettyHandler {
	return &PrettyHandler{
		Handler: slog.NewJSONHandler(out, opts.SlogOpts),
		l:       stdLog.New(out, "", 0),
	}
}

// SetupLogger создаёт *slog.Logger с подходящим обработчиком для указанного окружения.
func SetupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
		log.Warn(fmt.Sprintf("unknown env %q, falling back to prod settings", env))
	}

	slog.SetDefault(log)

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}
	handler := opts.NewPrettyHandler(os.Stdout)
	return slog.New(handler)
}

func levelColor(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return colorCyan
	case level < slog.LevelWarn:
		return colorGreen
	case level < slog.LevelError:
		return colorYellow
	default:
		return colorRed
	}
}

// Handle форматирует запись лога и выводит её в writer.
func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	color := levelColor(r.Level)
	level := color + r.Level.String() + ":" + colorReset

	prefix := ""
	if len(h.groups) > 0 {
		prefix = strings.Join(h.groups, ".") + "."
	}

	fields := make(map[string]any, r.NumAttrs())
	r.Attrs(func(a slog.Attr) bool {
		fields[prefix+a.Key] = a.Value.Any()
		return true
	})
	for _, a := range h.attrs {
		fields[prefix+a.Key] = a.Value.Any()
	}

	var b []byte
	var err error
	if len(fields) > 0 {
		b, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	}

	timeStr := r.Time.Format("[15:04:05.000]")
	msg := color + r.Message + colorReset

	h.l.Println(timeStr, level, msg, string(b))

	return nil
}

// WithAttrs возвращает новый обработчик с добавленными атрибутами.
func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{
		opts:    h.opts,
		groups:  h.groups,
		Handler: h.Handler,
		l:       h.l,
		attrs:   append(h.attrs, attrs...),
	}
}

// WithGroup возвращает новый обработчик с именем группы.
func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return &PrettyHandler{
		opts:    h.opts,
		groups:  append(h.groups, name),
		Handler: h.Handler.WithGroup(name),
		l:       h.l,
		attrs:   h.attrs,
	}
}
