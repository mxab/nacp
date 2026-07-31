package testutil

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type LogConsumer struct {
	mu      sync.RWMutex
	logs    []tc.Log
	stderrs []string
	stdouts []string
}

func (lc *LogConsumer) Accept(log tc.Log) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if log.LogType == tc.StdoutLog {
		lc.stdouts = append(lc.stdouts, string(log.Content))
	} else if log.LogType == tc.StderrLog {
		lc.stderrs = append(lc.stderrs, string(log.Content))
	} else {
		fmt.Printf("unknown log type: %s\n", log.LogType)
	}
	lc.logs = append(lc.logs, log)
}

func (lc *LogConsumer) Contains(message string) bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	for _, log := range lc.stderrs {
		if strings.Contains(log, message) {
			return true
		}
	}
	for _, log := range lc.stdouts {
		if strings.Contains(log, message) {
			return true
		}
	}
	return false
}

func LaunchCollector(t *testing.T) (tc.Container, *LogConsumer) {

	t.Helper()

	logConsumer := &LogConsumer{}
	req := tc.ContainerRequest{
		Image:        "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:0.133.0",
		ExposedPorts: []string{"4318/tcp"},

		Cmd: []string{
			`--config=yaml:receivers::otlp::protocols::http::endpoint: 0.0.0.0:4318`,
			`--config=yaml:service::pipelines::logs::receivers: [otlp]`,
			`--config=yaml:service::pipelines::logs::exporters: [debug]`,
			`--config=yaml:service::pipelines::metrics::receivers: [otlp]`,
			`--config=yaml:service::pipelines::metrics::exporters: [debug]`,
			`--config=yaml:service::pipelines::traces::receivers: [otlp]`,
			`--config=yaml:service::pipelines::traces::exporters: [debug]`,
			`--config=yaml:exporters::debug::verbosity: normal`},
		WaitingFor: wait.ForLog("Everything is ready. Begin running and processing data."),
		LogConsumerCfg: &tc.LogConsumerConfig{
			Opts:      []tc.LogProductionOption{tc.WithLogProductionTimeout(5 * time.Second)},
			Consumers: []tc.LogConsumer{logConsumer},
		},
	}

	ctx := t.Context()
	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}
	tc.CleanupContainer(t, c)
	url, err := c.PortEndpoint(ctx, "4318", "http")
	if err != nil {
		t.Fatalf("failed to get container endpoint: %v", err)
	}
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", url)
	os.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	os.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")

	t.Cleanup(func() {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		os.Unsetenv("OTEL_EXPORTER_OTLP_INSECURE")
		os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	})

	return c, logConsumer
}
