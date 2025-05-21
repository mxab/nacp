package testutil

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/mxab/nacp/admissionctrl/types"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/mock"
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

func (m *MockMutator) Mutate(payload *types.Payload) (out *api.Job, mutated bool, warnings []error, err error) {
	args := m.Called(payload)
	return args.Get(0).(*api.Job), args.Bool(1), args.Get(2).([]error), args.Error(3)
}
func (m *MockMutator) Name() string {
	return "mock-mutator"
}

type MockValidator struct {
	mock.Mock
}

func (m *MockValidator) Validate(payload *types.Payload) (warnings []error, err error) {
	args := m.Called(payload)
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
