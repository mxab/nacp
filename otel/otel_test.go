package otel

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
)

func TestHclog(t *testing.T) {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   "nacp",
		Level:  hclog.LevelFromString("DEBUG"),
		Output: os.Stdout,
	})
	logger.Debug("debug log entry", "key", nil)
}
func TestSlog(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("test log entry", "key", "value")
}

func TestOtlpSetup(t *testing.T) {

	logger := otelslog.NewLogger("mytest")

	_, logConsumer := testutil.LaunchCollector(t)

	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)

	otelShutdown, err := SetupOTelSDK(ctx, true, true, true, "0.0.0")
	if err != nil {
		t.Fatalf("failed to setup OTel SDK: %v", err)
	}

	mProvider := otel.GetMeterProvider()
	assert.NotNil(mProvider)

	logger.Info("some test log", "foo", "bar", "error", fmt.Errorf("test error"))

	err = otelShutdown(ctx)
	require.NoError(err, "failed to shutdown OTel SDK")

	time.Sleep(5 * time.Second)
	foundLogs := false

	for _, log := range logConsumer.Stderrs {

		if strings.Contains(log, "some test log foo=bar error=test error") {
			foundLogs = true
			break
		}
	}
	for _, log := range logConsumer.Stdouts {

		if strings.Contains(log, "some test log foo=bar error=test error") {
			foundLogs = true
			break
		}
	}
	assert.True(foundLogs, "expected log not found")
}
func TestOtlpSetupWith(t *testing.T) {

	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)

	lr, mr, se := testutil.OtelExporters(t)
	otelShutdown, flush, err := SetupOTelSDKWith(ctx, lr, mr, se)
	if err != nil {
		t.Fatalf("failed to setup OTel SDK: %v", err)
	}
	logger := otelslog.NewLogger("mytest")

	mProvider := otel.GetMeterProvider()
	assert.NotNil(mProvider)

	logger.Info("some test log", "foo", "bar", "error", fmt.Errorf("test error"))

	require.NoError(err, "failed to shutdown OTel SDK")

	flushErr := flush(ctx)
	require.NoError(flushErr, "failed to flush OTel SDK")

	err = otelShutdown(ctx)
	require.NoError(err, "failed to shutdown OTel SDK")

	wantLogs := logtest.Recording{
		logtest.Scope{
			Name: "mytest",
		}: []logtest.Record{
			{
				Context:      context.Background(),
				Severity:     log.SeverityInfo,
				SeverityText: log.SeverityInfo.String(),
				Body:         log.StringValue("some test log"),
				Attributes: []log.KeyValue{
					log.String("foo", "bar"),
					log.String("error", "test error"),
				},
			},
		},
	}
	logtest.AssertEqual(t, wantLogs, lr.Result(),
		// Ignore Timestamps.
		logtest.Transform(func(time.Time) time.Time {
			return time.Time{}
		}),
	)

}
