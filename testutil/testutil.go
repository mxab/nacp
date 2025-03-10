package testutil

import (
	"encoding/json"
	"github.com/mxab/nacp/admissionctrl/types"
	"io"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/mock"
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

func (m *MockMutator) Mutate(payload *types.Payload) (out *api.Job, warnings []error, err error) {
	args := m.Called(payload)
	return args.Get(0).(*api.Job), args.Get(1).([]error), args.Error(2)
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
