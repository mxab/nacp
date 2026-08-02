package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"github.com/mxab/nacp/pkg/admissionctrl/opa/bundle"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/config"
	"github.com/mxab/nacp/pkg/logutil"
	sdktest "github.com/open-policy-agent/opa/v1/sdk/test"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/logtest"
	metricSdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func readJobJson(t *testing.T, name string) []byte {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Could not get filename")
	}

	path := path.Join(path.Dir(filename), "..", "testdata", name)

	jsonFile, err := os.Open(path)

	if err != nil {
		t.Fatalf("Error opening file")
	}
	defer jsonFile.Close()
	data, err := io.ReadAll(jsonFile)
	if err != nil {
		t.Fatalf("Error reading file")
	}
	return data
}
func ReadJobJson(t *testing.T, name string) string {
	t.Helper()
	return string(readJobJson(t, name))
}
func ReadJob(t *testing.T, name string) *api.Job {
	t.Helper()

	data := readJobJson(t, name)
	job := &api.Job{}
	err := json.Unmarshal(data, &job)
	if err != nil {
		t.Fatalf("Error unmarshalling json")
	}
	return job
}

type MockMutator struct {
	mock.Mock
}

func (m *MockMutator) Mutate(ctx context.Context, payload *types.Payload) (out *api.Job, mutated bool, warnings []error, err error) {
	args := m.Called(ctx, payload)
	job := out
	if args.Get(0) != nil {
		job = args.Get(0).(*api.Job)
	}
	return job, args.Bool(1), args.Get(2).([]error), args.Error(3)
}
func (m *MockMutator) Name() string {
	return "mock-mutator"
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(ctx context.Context, payload *types.Payload) (warnings []error, err error) {
	args := m.Called(ctx, payload)
	return args.Get(0).([]error), args.Error(1)
}
func (m *MockValidator) Name() string {
	return "mock-validator"
}

func Filepath(t *testing.T, name string) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Could not get filename")
	}

	return path.Join(path.Dir(filename), "..", "testdata", name)
}

func OtelExporters(t *testing.T) (*logtest.Recorder, *metricSdk.ManualReader, *tracetest.InMemoryExporter) {
	t.Helper()
	spanExporter := tracetest.NewInMemoryExporter()
	logRecorder := logtest.NewRecorder()

	manualReader := metricSdk.NewManualReader()

	return logRecorder, manualReader, spanExporter
}
func MockValidatorReturningWarnings(warning string) *MockValidator {

	validator := new(MockValidator)
	validator.On("Validate", mock.Anything, mock.Anything).Return([]error{fmt.Errorf("%s", warning)}, nil)
	return validator
}

func MockValidatorReturningError(err string) *MockValidator {

	validator := new(MockValidator)
	validator.On("Validate", mock.Anything, mock.Anything).Return([]error{}, fmt.Errorf("%s", err))
	return validator
}
func MockMutatorReturningWarnings(warning string) *MockMutator {
	mutator := new(MockMutator)
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(BaseJob(), false, []error{fmt.Errorf("%s", warning)}, nil)
	return mutator
}
func MockMutatorReturningError(err string) *MockMutator {
	mutator := new(MockMutator)
	mutator.On("Mutate", mock.Anything, mock.Anything).Return(nil, false, []error{}, fmt.Errorf("%s", err))
	return mutator
}
func MockMutatorMutating(mutatedJob *api.Job) *MockMutator {
	mutator := new(MockMutator)

	mutator.On("Mutate", mock.Anything, mock.Anything).Return(mutatedJob, true, []error{}, nil)
	return mutator
}

func BaseJob() *api.Job {

	id := "test-job"
	job := &api.Job{
		ID: &id,
	}
	return job
}

// SetupOpa starts a bundle server hosting policy and returns a bundle Instance
// built through the same path production uses, so tests cover the real startup
// (readiness channel, slog logger, status listener) rather than a lookalike.
func SetupOpa(t *testing.T, policy string) *bundle.Instance {
	t.Helper()
	return SetupOpaBundles(t, map[string]string{"test": policy})[0]
}

// SetupOpaBundles starts one bundle server and one Instance per named policy.
func SetupOpaBundles(t *testing.T, policies map[string]string) []*bundle.Instance {
	t.Helper()

	registry := SetupOpaRegistry(t, policies)

	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)

	instances := make([]*bundle.Instance, 0, len(names))
	for _, name := range names {
		instance, err := registry.Get(name)
		require.NoError(t, err, "Bundle %q is registered", name)
		instances = append(instances, instance)
	}
	return instances
}

// SetupOpaRegistry builds a bundle Registry holding one Instance per named
// policy, each backed by its own mock bundle server.
func SetupOpaRegistry(t *testing.T, policies map[string]string) *bundle.Registry {
	t.Helper()

	names := make([]string, 0, len(policies))
	for name := range policies {
		names = append(names, name)
	}
	sort.Strings(names)

	configs := make([]config.OpaBundle, 0, len(names))
	for _, name := range names {
		configs = append(configs, config.OpaBundle{
			Id:         name,
			ConfigPath: writeOpaConfig(t, name, policies[name]),
		})
	}

	loggerFactory, _ := logutil.NewLoggerFactory(io.Discard, io.Discard, false)
	registry, stop, err := bundle.Setup(t.Context(), loggerFactory, configs)
	require.NoError(t, err, "No error setting up OPA bundles")
	t.Cleanup(stop)

	return registry
}

// writeOpaConfig starts a mock bundle server for policy and writes an OPA
// configuration pointing at it, returning the config path.
func writeOpaConfig(t *testing.T, name, policy string) string {
	t.Helper()

	server, err := sdktest.NewServer(sdktest.MockBundle("/bundles/bundle.tar.gz", map[string]string{
		"example.rego": policy,
	}))
	require.NoError(t, err, "No error creating mock server")
	t.Cleanup(server.Stop)

	opaConfig := fmt.Sprintf(`{
		"services": {%q: {"url": %q}},
		"bundles": {%q: {"service": %q, "resource": "/bundles/bundle.tar.gz"}},
		"decision_logs": {"console": true}
	}`, name, server.URL(), name, name)

	configPath := filepath.Join(t.TempDir(), name+".json")
	require.NoError(t, os.WriteFile(configPath, []byte(opaConfig), 0o600), "No error writing OPA config")
	return configPath
}
