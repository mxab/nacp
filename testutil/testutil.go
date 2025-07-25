package testutil

import (
	"context"
	"encoding/json"
	"fmt"
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
