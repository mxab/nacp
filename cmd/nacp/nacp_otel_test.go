package main

import (
	"compress/gzip"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/otel"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/logtest"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
)

func TestOtelInstrumentation(t *testing.T) {

	type test struct {
		name string

		requestSender func(*api.Client) (interface{}, *api.WriteMeta, error)

		nomadResponse         string
		nomadResponseEncoding string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator

		expectedMetricWithValue []map[string]metricdata.Aggregation

		wantLogs logtest.Recording
	}

	tests := []test{
		{

			name: "validator warning increments warning.count",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				testutil.MockValidatorReturningWarnings("some warning"),
			},

			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.validator.warning.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("validator.name", "mock-validator")),
								Value:      1,
							},
						},
					},
				},
			},
		},

		{

			name: "validator error increments error.count",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				testutil.MockValidatorReturningError("some error"),
			},

			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.validator.error.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("validator.name", "mock-validator")),
								Value:      1,
							},
						},
					},
				},
			},
		},
		{
			name: "mutator warning increments warning.count",
			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},
			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			mutators: []admissionctrl.JobMutator{
				testutil.MockMutatorReturningWarnings("some warning"),
			},
			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.mutator.warning.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("mutator.name", "mock-mutator")),
								Value:      1,
							},
						},
					},
				},
			},
		},
		{
			name: "mutator error increments error.count",
			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},
			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			mutators: []admissionctrl.JobMutator{
				testutil.MockMutatorReturningError("some error"),
			},
			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.mutator.error.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("mutator.name", "mock-mutator")),
								Value:      1,
							},
						},
					},
				},
			},
		},
		{
			name: "mutator mutating increments mutation.count",
			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},
			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			mutators: []admissionctrl.JobMutator{
				testutil.MockMutatorMutating(testutil.ReadJob(t, "job.json")), // we don't care about the mutated job
			},
			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.mutator.mutation.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("mutator.name", "mock-mutator")),
								Value:      1,
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			ctx := t.Context()

			logRecorder, metricReader, traceReader := testutil.OtelExporters(t)

			shutdown, flush, err := otel.SetupOTelSDKWith(ctx, logRecorder, metricReader, traceReader)
			require.NoError(t, err)

			slog.SetLogLoggerLevel(slog.LevelInfo)

			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

				if tc.nomadResponseEncoding == "gzip" {
					rw.Header().Set("Content-Encoding", "gzip")
					rw.WriteHeader(http.StatusOK)
					gzipWriter := gzip.NewWriter(rw)
					defer gzipWriter.Close()
					gzipWriter.Write([]byte(tc.nomadResponse))
				} else {
					rw.WriteHeader(http.StatusOK)
					rw.Write([]byte(tc.nomadResponse))
				}
			}))
			defer nomadDummy.Close()

			nomadURL, err := url.Parse(nomadDummy.URL)
			require.NoError(t, err)

			proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
			jobHandler := admissionctrl.NewJobHandler(
				tc.mutators,
				tc.validators,
				otelslog.NewLogger("testnacp"),
				false,
			)

			proxyHandlerFunc := NewProxyAsHandlerFunc(nomadURL, jobHandler, otelslog.NewLogger("testnacp"), proxyTransport)
			proxyServer := httptest.NewServer(proxyHandlerFunc)

			defer proxyServer.Close()
			nomadClient := buildNomadClient(t, proxyServer)

			tc.requestSender(nomadClient)

			resourceMetrics := &metricdata.ResourceMetrics{}
			require.NoError(t, metricReader.Collect(t.Context(), resourceMetrics))

			flush(ctx)

			spans := traceReader.GetSpans()

			require.NoError(t, shutdown(ctx))

			require.NotEmpty(t, resourceMetrics.ScopeMetrics, "Expected metrics to be found")

			AssertScopeMetricHasAttributes(t, resourceMetrics.ScopeMetrics, "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp", "http.server.request.body.size", attribute.String("http.request.method", "PUT"))
			AssertScopeMetricHasAttributes(t, resourceMetrics.ScopeMetrics, "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp", "http.server.request.duration", attribute.String("http.request.method", "PUT"))

			for _, expectedMetric := range tc.expectedMetricWithValue {
				for name, expected := range expectedMetric {
					AssertScopeMetricHasValue(t, resourceMetrics.ScopeMetrics, "nacp.controller", name, expected)
				}
			}

			require.NotEmpty(t, spans, "Expected spans to be found")
			//TODO: fix with reasonable assertion
			assert.GreaterOrEqual(t, len(spans), 4, "Expected at least 4 spans")

		})
	}
}

func AssertScopeMetricHasAttributes(t *testing.T, scopeMetrics []metricdata.ScopeMetrics, scope, metricName string, attrs ...attribute.KeyValue) bool {
	t.Helper()
	found := false
	for _, scopeMetric := range scopeMetrics {
		if scope == scopeMetric.Scope.Name {
			for _, metric := range scopeMetric.Metrics {
				if metric.Name == metricName {
					found = true
					metricdatatest.AssertHasAttributes(t, metric, attrs...)

				}
			}
		}
	}
	return assert.Truef(t, found, "Expected metric %s to be found in scope %s", metricName, scope)
}
func AssertScopeMetricHasValue(t *testing.T, scopeMetrics []metricdata.ScopeMetrics, scope, metricName string, expected metricdata.Aggregation) bool {
	t.Helper()

	for _, scopeMetric := range scopeMetrics {
		if scope == scopeMetric.Scope.Name {
			for _, metric := range scopeMetric.Metrics {
				if metric.Name == metricName {

					return metricdatatest.AssertAggregationsEqual(t, expected, metric.Data, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreExemplars())

				}
			}
		}
	}
	return assert.Failf(t, "metric not found", "metric with name %s not found in scope %s", metricName, scope)
}
func AssertLogBodyPresent(t *testing.T, records logtest.Recording, body string, attrs ...attribute.KeyValue) bool {
	t.Helper()
	found := false

	for _, logRecord := range records {
		for _, record := range logRecord {
			if body == record.Body.AsString() {

				found = true

				for _, a := range attrs {
					attrFound := false

					for _, kv := range record.Attributes {

						if kv.Key == string(a.Key) {
							attrFound = true
							assert.Equal(t, kv.Value.AsString(), a.Value.AsString(), "Expected log to contain %s", a)
							break
						}

					}
					assert.True(t, attrFound, "Expected log to '%s' contain %v = %v", body, a.Key, a.Value.AsString())

				}
			}
		}
	}
	return assert.True(t, found, "Expected log body to be present")
}

func rec(msgs ...logtest.Record) logtest.Recording {
	return logtest.Recording{
		logtest.Scope{
			Name: "testnacp",
		}: msgs,
	}
}
func info(msg string, attrs ...log.KeyValue) logtest.Record {

	return message(log.SeverityInfo, msg, attrs...)
}
func debug(msg string, attrs ...log.KeyValue) logtest.Record {

	return message(log.SeverityDebug, msg, attrs...)
}
func message(level log.Severity, msg string, attrs ...log.KeyValue) logtest.Record {

	return logtest.Record{
		Context:      context.Background(),
		Severity:     level,
		SeverityText: level.String(),
		Body:         log.StringValue(msg),
		Attributes:   attrs,
	}
}
func s(key, value string) log.KeyValue {
	return log.String(key, value)
}
func n(key string, value int) log.KeyValue {
	return log.Int(key, value)
}
func TestOtelLogInstrumentation(t *testing.T) {

	type test struct {
		name string

		requestSender func(*api.Client) (interface{}, *api.WriteMeta, error)

		nomadResponse         string
		nomadResponseEncoding string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator

		expectedMetricWithValue []map[string]metricdata.Aggregation

		wantLogs logtest.Recording
	}

	tests := []test{
		{

			name: "validator warning increments warning.count",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				testutil.MockValidatorReturningWarnings("some warning"),
			},

			expectedMetricWithValue: []map[string]metricdata.Aggregation{
				{
					"nacp.validator.warning.count": metricdata.Sum[float64]{
						Temporality: metricdata.CumulativeTemporality,
						IsMonotonic: true,
						DataPoints: []metricdata.DataPoint[float64]{
							{
								Attributes: attribute.NewSet(attribute.String("validator.name", "mock-validator")),
								Value:      1,
							},
						},
					},
				},
			},
			wantLogs: rec(
				info("Request received", s("clientIP", "127.0.0.1"), s("method", "PUT"), s("path", "/v1/jobs")),
				debug("applying job mutators", s("job", "example"), n("mutators", 0)),
				debug("applying job validators", s("job", "example"), n("validators", 1)),
				debug("applying job validator", s("job", "example"), s("validator", "mock-validator")),
				debug("job validate results", s("job", "example"), s("validator", "mock-validator"), log.Slice("warnings", log.StringValue("some warning")), log.Empty("error")),
				debug("Job after admission controllers", s("job", "{\"Submission\":null,\"Job\":{\"Region\":\"global\",\"Namespace\":\"default\",\"ID\":\"example\",\"Name\":\"example\",\"Type\":\"service\",\"Priority\":50,\"AllAtOnce\":false,\"Datacenters\":[\"dc1\"],\"NodePool\":null,\"Constraints\":null,\"Affinities\":null,\"TaskGroups\":[{\"Name\":\"cache\",\"Count\":1,\"Constraints\":null,\"Affinities\":null,\"Tasks\":[{\"Name\":\"redis\",\"Driver\":\"docker\",\"User\":\"\",\"Lifecycle\":null,\"Config\":{\"auth_soft_fail\":true,\"image\":\"redis:7\",\"ports\":[\"db\"]},\"Constraints\":null,\"Affinities\":null,\"Env\":null,\"Services\":null,\"Resources\":{\"CPU\":500,\"Cores\":0,\"MemoryMB\":256,\"MemoryMaxMB\":0,\"DiskMB\":0,\"Networks\":null,\"Devices\":null,\"NUMA\":null,\"SecretsMB\":null,\"IOPS\":0},\"RestartPolicy\":{\"Interval\":1800000000000,\"Attempts\":2,\"Delay\":15000000000,\"Mode\":\"fail\",\"RenderTemplates\":null},\"Meta\":null,\"KillTimeout\":5000000000,\"LogConfig\":{\"MaxFiles\":10,\"MaxFileSizeMB\":10,\"Enabled\":null,\"Disabled\":null},\"Artifacts\":null,\"Vault\":{\"Policies\":[\"example-redis\"],\"Role\":\"\",\"Namespace\":\"\",\"Cluster\":\"\",\"Env\":true,\"DisableFile\":null,\"ChangeMode\":\"restart\",\"ChangeSignal\":\"SIGHUP\",\"AllowTokenExpiration\":null},\"Consul\":null,\"Templates\":null,\"DispatchPayload\":null,\"VolumeMounts\":null,\"Leader\":false,\"ShutdownDelay\":0,\"KillSignal\":\"\",\"Kind\":\"\",\"ScalingPolicies\":null,\"Identity\":null,\"Identities\":null,\"Actions\":null,\"Schedule\":null}],\"Spreads\":null,\"Volumes\":null,\"RestartPolicy\":{\"Interval\":1800000000000,\"Attempts\":2,\"Delay\":15000000000,\"Mode\":\"fail\",\"RenderTemplates\":null},\"Disconnect\":null,\"ReschedulePolicy\":{\"Attempts\":0,\"Interval\":0,\"Delay\":30000000000,\"DelayFunction\":\"exponential\",\"MaxDelay\":3600000000000,\"Unlimited\":true},\"EphemeralDisk\":{\"Sticky\":false,\"Migrate\":false,\"SizeMB\":300},\"Update\":{\"Stagger\":30000000000,\"MaxParallel\":1,\"HealthCheck\":\"checks\",\"MinHealthyTime\":10000000000,\"HealthyDeadline\":300000000000,\"ProgressDeadline\":600000000000,\"Canary\":0,\"AutoRevert\":false,\"AutoPromote\":false},\"Migrate\":{\"MaxParallel\":1,\"HealthCheck\":\"checks\",\"MinHealthyTime\":10000000000,\"HealthyDeadline\":300000000000},\"Networks\":[{\"Mode\":\"\",\"Device\":\"\",\"CIDR\":\"\",\"IP\":\"\",\"DNS\":null,\"ReservedPorts\":null,\"DynamicPorts\":[{\"Label\":\"db\",\"Value\":0,\"To\":6379,\"HostNetwork\":\"default\",\"IgnoreCollision\":false}],\"Hostname\":\"\",\"MBits\":0,\"CNI\":null}],\"Meta\":null,\"Services\":null,\"ShutdownDelay\":null,\"StopAfterClientDisconnect\":null,\"MaxClientDisconnect\":null,\"Scaling\":null,\"Consul\":{\"Namespace\":\"\",\"Cluster\":\"\",\"Partition\":\"\"},\"PreventRescheduleOnLost\":null}],\"Update\":{\"Stagger\":30000000000,\"MaxParallel\":1,\"HealthCheck\":\"\",\"MinHealthyTime\":0,\"HealthyDeadline\":0,\"ProgressDeadline\":0,\"Canary\":0,\"AutoRevert\":false,\"AutoPromote\":false},\"Multiregion\":null,\"Spreads\":null,\"Periodic\":null,\"ParameterizedJob\":null,\"Reschedule\":null,\"Migrate\":null,\"Meta\":null,\"UI\":null,\"Stop\":false,\"ParentID\":\"\",\"Dispatched\":false,\"DispatchIdempotencyToken\":\"\",\"Payload\":null,\"ConsulNamespace\":\"\",\"VaultNamespace\":\"\",\"NomadTokenID\":\"\",\"Status\":\"pending\",\"StatusDescription\":\"\",\"Stable\":false,\"Version\":0,\"SubmitTime\":1675891187313750000,\"CreateIndex\":11,\"ModifyIndex\":11,\"JobModifyIndex\":11,\"VersionTag\":null},\"Region\":\"\",\"Namespace\":\"\",\"SecretID\":\"\"}")),
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			ctx := t.Context()

			logRecorder, metricReader, traceReader := testutil.OtelExporters(t)

			shutdown, flush, err := otel.SetupOTelSDKWith(ctx, logRecorder, metricReader, traceReader)
			require.NoError(t, err)

			slog.SetLogLoggerLevel(slog.LevelInfo)

			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

				if tc.nomadResponseEncoding == "gzip" {
					rw.Header().Set("Content-Encoding", "gzip")
					rw.WriteHeader(http.StatusOK)
					gzipWriter := gzip.NewWriter(rw)
					defer gzipWriter.Close()
					gzipWriter.Write([]byte(tc.nomadResponse))
				} else {
					rw.WriteHeader(http.StatusOK)
					rw.Write([]byte(tc.nomadResponse))
				}
			}))
			defer nomadDummy.Close()

			nomadURL, err := url.Parse(nomadDummy.URL)
			require.NoError(t, err)

			proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
			jobHandler := admissionctrl.NewJobHandler(
				tc.mutators,
				tc.validators,
				otelslog.NewLogger("testnacp"),
				false,
			)

			proxy := NewProxyAsHandlerFunc(nomadURL, jobHandler, otelslog.NewLogger("testnacp"), proxyTransport)
			proxyServer := httptest.NewServer(proxy)

			defer proxyServer.Close()
			nomadClient := buildNomadClient(t, proxyServer)

			tc.requestSender(nomadClient)

			flush(ctx)

			require.NoError(t, shutdown(ctx))

			logs := logRecorder.Result()

			logtest.AssertEqual(t, tc.wantLogs, logs,
				// Ignore Timestamps.
				logtest.Transform(func(r logtest.Record) logtest.Record {
					cp := r.Clone()
					cp.Context = nil           // Ignore context for comparison.
					cp.Timestamp = time.Time{} // Ignore timestamp for comparison.
					return cp
				}),
			)

			require.NotEmpty(t, logs, "Expected logs to be found")

		})
	}
}
