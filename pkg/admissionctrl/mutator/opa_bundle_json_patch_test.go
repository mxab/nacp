package mutator

import (
	"log/slog"
	"testing"

	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpaBundleMutatorName(t *testing.T) {

	opa := testutil.SetupOpa(t, "package mypolicy")
	mutator, err := NewOpaBundleMutator("test", "test/path", slog.Default(), opa)

	require.NoError(t, err, "No error creating mutator")

	assert.Equal(t, "test", mutator.Name(), "Name is correct")
}

func TestOpaBundleMutator(t *testing.T) {

	tt := []struct {
		name     string
		policy   string
		path     string
		inputJob *api.Job

		expectedJob     *api.Job
		expectedMutated bool
		expectedWarns   []string
		expectedErrs    []string
	}{
		{
			name: "no changes",
			policy: `package mypolicy
			`,
			path:            "/mypolicy",
			inputJob:        testutil.BaseJob(),
			expectedJob:     testutil.BaseJob(),
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{},
		},
		{
			name: "handle policy eval error",
			policy: `package mypolicy
			`,
			path:            "/nonexistentpath",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"failed to perform policy decision"},
		},
		{
			name: "add meta",
			policy: `package mypolicy
			patch = [
			{"op": "add", "path": "/Meta", "value": {}},
			{"op": "add", "path": "/Meta/hello", "value": "world"}
			]
			`,
			path:     "/mypolicy",
			inputJob: &api.Job{},
			expectedJob: &api.Job{
				Meta: map[string]string{
					"hello": "world",
				},
			},
			expectedMutated: true,
			expectedWarns:   []string{},
			expectedErrs:    []string{},
		},
		{
			name: "handle errors",
			policy: `package mypolicy
			errors = ["This is an error message", "This is another error message"]
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"This is an error message", "This is another error message"},
		},
		{
			name: "handle warnings",
			policy: `package mypolicy
			warnings = ["This is a warning message", "This is another warning message"]
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     &api.Job{},
			expectedMutated: false,
			expectedWarns:   []string{"This is a warning message", "This is another warning message"},
			expectedErrs:    []string{},
		},
		{
			name: "handle patch and warnings",
			policy: `package mypolicy
			patch = [
				{"op": "add", "path": "/Meta", "value": {}},
				{"op": "add", "path": "/Meta/hello", "value": "world"}
			]
			warnings = ["This is a warning message", "This is another warning message"]
			`,
			path:     "/mypolicy",
			inputJob: &api.Job{},
			expectedJob: &api.Job{
				Meta: map[string]string{
					"hello": "world",
				},
			},
			expectedMutated: true,
			expectedWarns:   []string{"This is a warning message", "This is another warning message"},
			expectedErrs:    []string{},
		},
		{
			name: "handle invalid error type",
			policy: `package mypolicy
			errors = 5
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"policy yielded an invalid errors value"},
		},
		{
			name: "handle invalid warning type as error",
			policy: `package mypolicy
			warnings = 5
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"policy yielded an invalid warnings value"},
		},
		{
			name: "handle invalid error entry type as error",
			policy: `package mypolicy
			errors = ["this is fine", 5]
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"policy yielded an invalid error entry value"},
		},
		{
			name: "handle invalid warning entry type as warning",
			policy: `package mypolicy
			warnings = ["this is fine", 5]
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{"this is fine"},
			expectedErrs:    []string{"policy yielded an invalid warning entry value"},
		},
		{
			name: "handle invalid patch type as error",
			policy: `package mypolicy
			patch = 5
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"policy yielded an invalid patch value"},
		},
		{
			name: "handle invalid patch entry type as error",
			policy: `package mypolicy
			patch = [5]
			`,
			path:            "/mypolicy",
			inputJob:        &api.Job{},
			expectedJob:     nil,
			expectedMutated: false,
			expectedWarns:   []string{},
			expectedErrs:    []string{"policy yielded patch failed"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opa := testutil.SetupOpa(t, tc.policy)
			mutator, err := NewOpaBundleMutator(tc.name, tc.path, slog.New(slog.DiscardHandler), opa)

			require.NoError(t, err, "No error creating mutator")
			result, mutated, warns, err := mutator.Mutate(t.Context(), tc.inputJob)

			assert.Equal(t, tc.expectedJob, result, "Job is correct")
			assert.Equal(t, tc.expectedMutated, mutated, "Mutated is correct")
			// warns
			assert.Len(t, warns, len(tc.expectedWarns), "Warnings length is correct")

			for i, expectedWarn := range tc.expectedWarns {
				assert.ErrorContains(t, warns[i], expectedWarn, "Warning is correct")
			}
			// errs
			if len(tc.expectedErrs) == 0 {
				assert.NoError(t, err, "No error")
			} else {
				for _, expectedErr := range tc.expectedErrs {
					assert.ErrorContains(t, err, expectedErr, "Error is correct")
				}
			}

		})
	}
}
