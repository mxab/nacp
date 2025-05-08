package otel

import (
	"bufio"
	"encoding/json"
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
func TestStdoutSetup(t *testing.T) {

	logger := otelslog.NewLogger("mytest")

	assert := assert.New(t)
	require := require.New(t)

	ctx := t.Context()
	_, _, _, otelShutdown, err := SetupOTelSDKWithInmemoryOutput(ctx)
	if err != nil {
		t.Fatalf("failed to setup OTel SDK: %v", err)
	}

	mProvider := otel.GetMeterProvider()
	assert.NotNil(mProvider)

	logger.Info("some test log", "foo", "bar", "error", fmt.Errorf("test error"))

	err = otelShutdown(ctx)
	require.NoError(err, "failed to shutdown OTel SDK")

}
func TestSetupWithReader(t *testing.T) {

	os.Setenv("OTEL_SERVICE_NAME", "nacptest")

	assertResource := func(signal map[string]interface{}) {
		resource, ok := signal["Resource"].([]interface{})

		if assert.True(t, ok, "Resource field is not a map") {
			assert.Contains(t, resource, map[string]interface{}{
				"Key":   "service.name",
				"Value": map[string]interface{}{"Type": "STRING", "Value": "nacptest"},
			}, "expected service.name not found")
		}

	}

	logger := otelslog.NewLogger("mytest")

	assert := assert.New(t)
	require := require.New(t)

	ctx := t.Context()
	lr, mr, tr, otelShutdown, err := SetupOTelSDKWithInmemoryOutput(ctx)
	if err != nil {
		t.Fatalf("failed to setup OTel SDK: %v", err)
	}

	mProvider := otel.GetMeterProvider()
	assert.NotNil(mProvider)

	logger.Info("some test log", "foo", "bar", "error", fmt.Errorf("test error"))

	err = otelShutdown(ctx)
	time.Sleep(5 * time.Second)
	require.NoError(err, "failed to shutdown OTel SDK")
	time.Sleep(5 * time.Second)
	logScanner := bufio.NewScanner(lr)
	logs := make([]map[string]interface{}, 0)
	for logScanner.Scan() {
		line := logScanner.Text()
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(line), &data)
		if assert.NoError(err) {
			logs = append(logs, data)
		}

	}

	if assert.Equal(1, len(logs), "expected log not found") {

		log := logs[0]
		assert.EqualValues(
			map[string]interface{}{
				"Type":  "String",
				"Value": "some test log",
			},
			log["Body"],
			"expected log not found",
		)

		assertResource(log)
	}

	metricScanner := bufio.NewScanner(mr)
	metrics := make([]map[string]interface{}, 0)
	for metricScanner.Scan() {
		line := metricScanner.Text()
		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(line), &data)
		if assert.NoError(err) {
			metrics = append(metrics, data)
		}
	}
	if assert.Equal(1, len(metrics), "expected metric not found") {

		assertResource(metrics[0])

		fmt.Println("metrics", metrics[0])

	}

	traceScanner := bufio.NewScanner(tr)
	traces := make([]map[string]interface{}, 0)
	for traceScanner.Scan() {
		line := traceScanner.Text()

		data := make(map[string]interface{})
		err := json.Unmarshal([]byte(line), &data)
		if assert.NoError(err) {
			traces = append(traces, data)
		}
	}
	assert.Equal(len(traces), 0, "expected trace not found")

	assert.True(logScanner.Err() == nil, "expected log not found")
	assert.True(metricScanner.Err() == nil, "expected metric not found")
	assert.True(traceScanner.Err() == nil, "expected trace not found")

}
func TestOtlpSetup(t *testing.T) {

	logger := otelslog.NewLogger("mytest")

	_, logConsumer := testutil.LaunchCollector(t)

	ctx := t.Context()
	assert := assert.New(t)
	require := require.New(t)

	otelShutdown, err := SetupOTelSDK(ctx, true, true, true)
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
