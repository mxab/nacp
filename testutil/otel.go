package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type LogConsumer struct {
	Logs []tc.Log

	Stderrs []string
	Stdouts []string
}

func (lc *LogConsumer) Accept(log tc.Log) {

	//fmt.Println(string(log.Content))

	if log.LogType == tc.StdoutLog {
		lc.Stdouts = append(lc.Stdouts, string(log.Content))
	} else if log.LogType == tc.StderrLog {
		lc.Stderrs = append(lc.Stderrs, string(log.Content))
	} else {
		fmt.Printf("unknown log type: %s\n", log.LogType)
	}
	lc.Logs = append(lc.Logs, log)
}

func LaunchCollector(t *testing.T) (tc.Container, *LogConsumer) {

	t.Helper()

	logConsumer := &LogConsumer{}
	req := tc.ContainerRequest{
		Image:        "ghcr.io/open-telemetry/opentelemetry-collector-releases/opentelemetry-collector:0.123.0",
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
