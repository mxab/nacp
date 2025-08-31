package logutil

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/mxab/nacp/config"
	slogmulti "github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/processors/minsev"
)

type Level string
type SlogOut string
type SlogFormat string

const (
	Error Level = "ERROR"
	Warn  Level = "WARN"
	Info  Level = "INFO"
	Debug Level = "DEBUG"

	SlogFormatJson SlogFormat = "json"
	SlogFormatText SlogFormat = "text"
)

func (l Level) convert() (slog.Level, minsev.Severity) {

	switch l {
	case Error:
		return slog.LevelError, minsev.SeverityError
	case Warn:
		return slog.LevelWarn, minsev.SeverityWarn
	case Info:
		return slog.LevelInfo, minsev.SeverityInfo
	case Debug:
		return slog.LevelDebug, minsev.SeverityDebug
	default:
		panic("unknown level")
	}

}

type Leveler struct {
	slogVar   *slog.LevelVar
	minsevVar *minsev.SeverityVar
}

func NewLeveler(initial Level) *Leveler {
	lev, sev := initial.convert()
	slogVar := slog.LevelVar{}
	slogVar.Set(lev)
	minsevVar := minsev.SeverityVar{}
	minsevVar.Set(sev)
	return &Leveler{
		slogVar:   &slogVar,
		minsevVar: &minsevVar,
	}
}
func (l *Leveler) Set(level Level) {
	lev, sev := level.convert()
	l.slogVar.Set(lev)
	l.minsevVar.Set(sev)
}
func (l *Leveler) GetSlogLeveler() slog.Leveler {
	return l.slogVar
}
func (l *Leveler) GetSeverietier() minsev.Severitier {
	return l.minsevVar
}

type LoggerFactory struct {
	leveler *Leveler
	jsonOut io.Writer
	textOut io.Writer
	otel    bool
}

func NewLoggerFactory(jsonOut io.Writer, textOut io.Writer, otel bool) (*LoggerFactory, *Leveler) {
	leveler := NewLeveler(Info)
	return &LoggerFactory{
		leveler: leveler,
		jsonOut: jsonOut,
		textOut: textOut,
		otel:    otel,
	}, leveler
}
func outStrToWriter(out string) io.Writer {
	switch out {
	case "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		return nil
	}
}
func NewLoggerFactoryFromConfig(logging *config.Logging) (*LoggerFactory, *Leveler) {

	leveler := NewLeveler(Level(strings.ToUpper(logging.Level)))
	var textOut io.Writer
	if *logging.SlogLogging.Text {
		textOut = outStrToWriter(*logging.SlogLogging.TextOut)
	}
	var jsonOut io.Writer
	if *logging.SlogLogging.Json {
		jsonOut = outStrToWriter(*logging.SlogLogging.JsonOut)
	}

	return &LoggerFactory{
		textOut: textOut,
		jsonOut: jsonOut,
		otel:    *logging.OtelLogging.Enabled,
		leveler: leveler,
	}, leveler

}

func (lf *LoggerFactory) GetLogger(name string) *slog.Logger {

	var handlers []slog.Handler

	if lf.jsonOut != nil {
		slogHandler := slog.NewJSONHandler(lf.jsonOut, &slog.HandlerOptions{
			Level: lf.leveler.GetSlogLeveler(),
		})
		handlers = append(handlers, slogHandler)
	}
	if lf.textOut != nil {
		textHandler := slog.NewTextHandler(lf.textOut, &slog.HandlerOptions{
			Level: lf.leveler.GetSlogLeveler(),
		})
		handlers = append(handlers, textHandler)
	}
	if lf.otel {
		otelHandler := otelslog.NewHandler(name)
		handlers = append(handlers, otelHandler)
	}

	return slog.New(slogmulti.Fanout(handlers...))
}
