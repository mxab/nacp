package testutil

import (
	"encoding/json"
	"io"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/hashicorp/nomad/nomad/structs"
)

func ReadJob(t *testing.T) *structs.Job {
	t.Helper()
	job := &structs.Job{}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("Could not get filename")
	}

	path := path.Join(path.Dir(filename), "..", "testdata", "job.json")

	jsonFile, err := os.Open(path)

	if err != nil {
		t.Fatalf("Error opening file")
	}
	defer jsonFile.Close()
	data, err := io.ReadAll(jsonFile)
	if err != nil {
		t.Fatalf("Error reading file")
	}
	err = json.Unmarshal(data, &job)
	if err != nil {
		t.Fatalf("Error unmarshalling json")
	}
	return job
}
