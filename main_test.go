package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// rewrite the test above as table driven test
func TestProxyTableDriven(t *testing.T) {

	type test struct {
		name                  string
		path                  string
		method                string
		requestJson           string
		wantNomadRequestJson  string
		wantProxyResponseJson string
		nomadResponse         string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator
	}

	tests := []test{
		{
			name:   "create job adds hello meta",
			path:   "/v1/jobs",
			method: "PUT",

			requestJson:          registerRequestJson(t, testutil.ReadJob(t, "job.json")),
			wantNomadRequestJson: registerRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{}),
			nomadResponse:         toJson(t, &api.JobRegisterResponse{}),
			validators:            []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},
		{
			name:        "update job adds hello meta",
			path:        "/v1/job/some-job",
			method:      "PUT",
			requestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantNomadRequestJson:  registerRequestJson(t, jobWithHelloWorldMeta(t)),
			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{}),

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},
		{
			name:                 "plan job adds hello meta",
			path:                 "/v1/job/some-job/plan",
			method:               "PUT",
			requestJson:          planRequestJson(t, testutil.ReadJob(t, "job.json")),
			wantNomadRequestJson: planRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{}),
			nomadResponse:         toJson(t, &api.JobRegisterResponse{}),

			validators: []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},
		{
			name:        "create job adds warnings",
			path:        "/v1/jobs",
			method:      "PUT",
			requestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),
			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{
				Warnings: "1 error occurred:\n\t* some warning\n\n",
			}),

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:        "create job concats warnings",
			path:        "/v1/jobs",
			method:      "PUT",
			requestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),
			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{
				Warnings: "2 errors occurred:\n\t* 1 error occurred:\n\t* some warning\n\n\n\t* some warning\n\n",
			}),

			nomadResponse: toJson(t, &api.JobRegisterResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:        "plan job adds warnings",
			path:        "/v1/job/some-job/plan",
			method:      "PUT",
			requestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantNomadRequestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),
			wantProxyResponseJson: toJson(t, &api.JobPlanResponse{
				// Diff: &api.JobDiff{
				// 	Fields: []*api.FieldDiff{
				// 		{
				// 			Name: "aaa",
				// 			Type: "bbb",
				// 			Old:  "ccc",
				// 			New:  "ddd",
				// 		},
				// 	},
				// },
				Warnings: "1 error occurred:\n\t* some warning\n\n",
			}),

			nomadResponse: toJson(t, &api.JobPlanResponse{
				// Diff: &api.JobDiff{
				// 	Fields: []*api.FieldDiff{
				// 		{
				// 			Name: "aaa",
				// 			Type: "bbb",
				// 			Old:  "ccc",
				// 			New:  "ddd",
				// 		},
				// 	},
				// },
			}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters

				assert.Equal(t, req.Method, tc.method, "Ensure method is set")
				assert.Equal(t, req.URL.Path, tc.path, "Ensure path is set")
				jsonData, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				json := string(jsonData)
				assert.JSONEq(t, tc.wantNomadRequestJson, json, "Body matches")

				_, _ = rw.Write([]byte(tc.nomadResponse))
			}))
			// Close the server when test finishes
			defer nomadDummy.Close()

			// Use Client & URL from our local test server

			nomad, err := url.Parse(nomadDummy.URL)
			if err != nil {
				t.Fatal(err)
			}
			jobHandler := admissionctrl.NewJobHandler(
				tc.mutators,
				tc.validators,
				hclog.NewNullLogger(),
			)
			proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger())

			proxyServer := httptest.NewServer(http.HandlerFunc(proxy))
			defer proxyServer.Close()

			res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, tc.path), strings.NewReader(tc.requestJson))
			assert.NoError(t, err, "No http call error")
			assert.Equal(t, 200, res.StatusCode, "OK response is expected")
			assert.JSONEq(t, tc.wantProxyResponseJson, readClosterToString(t, res.Body), "Body matches")
		})
	}

}
func mockValidatorReturningWarnings(warning string) admissionctrl.JobValidator {

	validator := new(testutil.MockValidator)
	validator.On("Validate", mock.Anything).Return([]error{fmt.Errorf(warning)}, nil)
	return validator

}
func readClosterToString(t *testing.T, rc io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func jobWithHelloWorldMeta(t *testing.T) *api.Job {
	wantJob := testutil.ReadJob(t, "job.json")
	wantJob.Meta = map[string]string{
		"hello": "world",
	}
	return wantJob
}
func toJson(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func registerRequestJson(t *testing.T, wantJob *api.Job) string {
	t.Helper()
	register := &api.JobRegisterRequest{
		Job: wantJob,
	}
	return toJson(t, register)

}

func planRequestJson(t *testing.T, wantJob *api.Job) string {
	t.Helper()
	plan := &api.JobPlanRequest{
		Job: wantJob,
	}
	return toJson(t, plan)

}

func TestAdmissionControllerErrors(t *testing.T) {
	nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		_, _ = rw.Write([]byte(`you should not see this`))
	}))
	// Close the server when test finishes
	defer nomadDummy.Close()

	validator := new(testutil.MockValidator)
	validator.On("Validate", mock.Anything).Return([]error{}, fmt.Errorf("some error"))

	nomad, err := url.Parse(nomadDummy.URL)
	if err != nil {
		t.Fatal(err)
	}
	jobHandler := admissionctrl.NewJobHandler(
		[]admissionctrl.JobMutator{},
		[]admissionctrl.JobValidator{validator},
		hclog.NewNullLogger(),
	)
	proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger())

	proxyServer := httptest.NewServer(http.HandlerFunc(proxy))

	defer proxyServer.Close()

	jobRequestJson := registerRequestJson(t, testutil.ReadJob(t, "job.json"))
	res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, "/v1/jobs"), strings.NewReader(jobRequestJson))
	require.NoError(t, err, "No http call error")
	assert.Equal(t, 500, res.StatusCode, "Should return 400")
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	x := string(body)
	t.Logf("response: %s", x)
}

func sendPut(t *testing.T, url string, body io.Reader) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}
