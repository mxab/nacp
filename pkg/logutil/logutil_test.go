package logutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"

	"github.com/mxab/nacp/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockExporter struct {
	mock.Mock
}

func (m *MockExporter) Export(ctx context.Context, records []log.Record) error {
	args := m.Called(ctx, records)
	return args.Error(0)
}
func (m *MockExporter) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *MockExporter) ForceFlush(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestFilterLevel(t *testing.T) {

	type Messages []struct {
		level slog.Level
		input string
	}
	messages := Messages{

		{slog.LevelDebug, "This is a debug message"},
		{slog.LevelInfo, "This is an info message"},
		{slog.LevelWarn, "This is a warning message"},
		{slog.LevelError, "This is an error message"},
	}
	tests := []struct {
		level            slog.Level
		inputs           Messages
		expectedMessages Messages
	}{
		{
			level:            slog.LevelDebug,
			inputs:           messages,
			expectedMessages: messages,
		},
		{
			level:  slog.LevelInfo,
			inputs: messages,
			expectedMessages: Messages{
				{slog.LevelInfo, "This is an info message"},
				{slog.LevelWarn, "This is a warning message"},
				{slog.LevelError, "This is an error message"},
			},
		},
		{
			level:  slog.LevelWarn,
			inputs: messages,
			expectedMessages: Messages{
				{slog.LevelWarn, "This is a warning message"},
				{slog.LevelError, "This is an error message"},
			},
		},
		{
			level:  slog.LevelError,
			inputs: messages,
			expectedMessages: Messages{
				{slog.LevelError, "This is an error message"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s", tt.level), func(t *testing.T) {

			levler := NewLeveler(Debug)
			levler.Set(Level(tt.level.String()))

			jsonOut := &bytes.Buffer{}
			textOut := &bytes.Buffer{}
			otelMockExporter, loggerProvider := setup(levler)
			defer loggerProvider.Shutdown(t.Context())
			lf := &LoggerFactory{
				leveler: levler,
				jsonOut: jsonOut,
				textOut: textOut,
				otel:    true,
			}

			logger := lf.GetLogger("dummy")

			for _, msg := range tt.inputs {
				logger.Log(context.Background(), msg.level, msg.input)
			}
			loggerProvider.ForceFlush(t.Context())

			records := otelMockExporter.Calls[0].Arguments.Get(1).([]log.Record)
			assert.Len(t, records, len(tt.expectedMessages))
			jsonOutput := jsonOut.String()
			textOutput := textOut.String()
			assert.Len(t, strings.Split(strings.TrimSpace(jsonOutput), "\n"), len(tt.expectedMessages))
			assert.Len(t, strings.Split(strings.TrimSpace(textOutput), "\n"), len(tt.expectedMessages))

			for _, expected := range tt.expectedMessages {

				assert.Contains(t, jsonOutput, fmt.Sprintf("\"msg\":\"%s\"", expected.input))
				assert.Contains(t, textOutput, fmt.Sprintf("msg=\"%s\"", expected.input))
			}

		})

	}
}

func TestLevelChange(t *testing.T) {
	levler := NewLeveler(Debug)

	buf := &bytes.Buffer{}
	otelMockExporter, loggerProvider := setup(levler)
	defer loggerProvider.Shutdown(t.Context())
	lf := &LoggerFactory{
		leveler: levler,
		jsonOut: buf,
		otel:    true,
	}

	logger := lf.GetLogger("dummy")

	logger.Log(context.Background(), slog.LevelDebug, "This is a debug message")
	logger.Log(context.Background(), slog.LevelInfo, "This is an info message")
	logger.Log(context.Background(), slog.LevelWarn, "This is a warning message")
	logger.Log(context.Background(), slog.LevelError, "This is an error message")

	levler.Set(Warn)

	logger.Log(context.Background(), slog.LevelDebug, "This is a debug message")
	logger.Log(context.Background(), slog.LevelInfo, "This is an info message")
	logger.Log(context.Background(), slog.LevelWarn, "This is a warning message")
	logger.Log(context.Background(), slog.LevelError, "This is an error message")

	loggerProvider.ForceFlush(t.Context())

	records := otelMockExporter.Calls[0].Arguments.Get(1).([]log.Record)
	assert.Len(t, records, 6)
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 6)
	assert.Contains(t, output, "This is a debug message")
	assert.Contains(t, output, "This is an info message")
	assert.Contains(t, output, "This is a warning message")
	assert.Contains(t, output, "This is an error message")

	countDebug := 0
	countInfo := 0
	countWarn := 0
	countError := 0
	for _, line := range lines {
		if strings.Contains(line, "This is a debug message") {
			countDebug++
		}
		if strings.Contains(line, "This is an info message") {
			countInfo++
		}
		if strings.Contains(line, "This is a warning message") {
			countWarn++
		}
		if strings.Contains(line, "This is an error message") {
			countError++
		}
	}
	assert.Equal(t, 1, countDebug)
	assert.Equal(t, 1, countInfo)
	assert.Equal(t, 2, countWarn)
	assert.Equal(t, 2, countError)

}

func setup(levler *Leveler) (mockExporter *MockExporter, loggerProvider *log.LoggerProvider) {

	mockExporter = &MockExporter{}
	mockExporter.On("Export", mock.Anything, mock.Anything).Return(nil)
	mockExporter.On("Shutdown", mock.Anything).Return(nil)
	mockExporter.On("ForceFlush", mock.Anything).Return(nil)
	batchProcessor := log.NewBatchProcessor(mockExporter)
	processor := minsev.NewLogProcessor(batchProcessor, levler.GetSeverietier())
	loggerProvider = log.NewLoggerProvider(log.WithProcessor(processor))

	global.SetLoggerProvider(loggerProvider)
	return
}

func TestNewLoggerFactoryFromConfig(t *testing.T) {

	tt := []struct {
		name            string
		config          *config.Logging
		expectedTextOut io.Writer
		expectedJsonOut io.Writer
		expectedOtel    bool
	}{
		{
			name: "only text",
			config: &config.Logging{
				Level: "info",
				SlogLogging: &config.SlogLogging{
					Text:    config.Ptr(true),
					TextOut: config.Ptr("stdout"),
					Json:    config.Ptr(false),
					JsonOut: config.Ptr("stdout"),
				},
				OtelLogging: &config.OtelLogging{
					Enabled: config.Ptr(false),
				},
			},
			expectedTextOut: os.Stdout,
			expectedJsonOut: nil,
			expectedOtel:    false,
		},
		{
			name: "only text on stderr",
			config: &config.Logging{
				Level: "info",
				SlogLogging: &config.SlogLogging{
					Text:    config.Ptr(true),
					TextOut: config.Ptr("stderr"),
					Json:    config.Ptr(false),
					JsonOut: config.Ptr("stdout"),
				},
				OtelLogging: &config.OtelLogging{
					Enabled: config.Ptr(false),
				},
			},
			expectedTextOut: os.Stderr,
			expectedJsonOut: nil,
			expectedOtel:    false,
		},
		{
			name: "text and json",
			config: &config.Logging{
				Level: "info",
				SlogLogging: &config.SlogLogging{
					Text:    config.Ptr(true),
					TextOut: config.Ptr("stdout"),
					Json:    config.Ptr(true),
					JsonOut: config.Ptr("stderr"),
				},
				OtelLogging: &config.OtelLogging{
					Enabled: config.Ptr(false),
				},
			},
			expectedTextOut: os.Stdout,
			expectedJsonOut: os.Stderr,
			expectedOtel:    false,
		},

		{
			name: "only json to stdout",
			config: &config.Logging{
				Level: "info",
				SlogLogging: &config.SlogLogging{
					Text:    config.Ptr(false),
					TextOut: config.Ptr("stdout"),
					Json:    config.Ptr(true),
					JsonOut: config.Ptr("stdout"),
				},
				OtelLogging: &config.OtelLogging{
					Enabled: config.Ptr(false),
				},
			},
			expectedTextOut: nil,
			expectedJsonOut: os.Stdout,
			expectedOtel:    false,
		},
		{
			name: "only otel",
			config: &config.Logging{
				Level: "info",
				SlogLogging: &config.SlogLogging{
					Text:    config.Ptr(false),
					TextOut: config.Ptr("stdout"),
					Json:    config.Ptr(false),
					JsonOut: config.Ptr("stdout"),
				},
				OtelLogging: &config.OtelLogging{
					Enabled: config.Ptr(true),
				},
			},
			expectedTextOut: nil,
			expectedJsonOut: nil,
			expectedOtel:    true,
		},
	}

	for _, tt := range tt {
		t.Run(tt.name, func(t *testing.T) {
			lf, leveler := NewLoggerFactoryFromConfig(tt.config)

			assert.NotNil(t, lf)
			assert.NotNil(t, leveler)

			assert.True(t, tt.expectedTextOut == lf.textOut)
			assert.True(t, tt.expectedJsonOut == lf.jsonOut)
			assert.Equal(t, tt.expectedOtel, lf.otel)
		})
	}
}
