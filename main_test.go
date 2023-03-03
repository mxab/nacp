package main

import (
	"encoding/json"
	"errors"
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
	"github.com/hashicorp/nomad/helper"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// rewrite the test above as table driven test
func TestProxy(t *testing.T) {

	type test struct {
		name   string
		path   string
		method string

		requestSender        func(*api.Client) (interface{}, *api.WriteMeta, error)
		wantNomadRequestJson string
		wantProxyResponse    interface{}
		nomadResponse        string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator
	}

	tests := []test{

		{
			name:   "create job adds hello meta",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},
			wantNomadRequestJson: registerRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponse: &api.JobRegisterResponse{},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},

		{
			name:   "plan job adds hello meta",
			path:   "/v1/job/example/plan",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponse: &api.JobPlanResponse{},

			nomadResponse: toJson(t, &api.JobPlanResponse{}),

			validators: []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},
		{
			name:   "create job adds warnings",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobRegisterResponse{
				Warnings: "1 warning:\n\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "create job concats warnings",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobRegisterResponse{
				// TODO: rework error concatination
				Warnings: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "plan job adds warnings",
			path:   "/v1/job/example/plan",
			method: "PUT",
			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobPlanResponse{
				Warnings: "1 warning:\n\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobPlanResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "validate job adds hello meta",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(testutil.ReadJob(t, "job.json"), nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: jobWithHelloWorldMeta(t)}),

			wantProxyResponse: &api.JobValidateResponse{},

			nomadResponse: toJson(t, &api.JobValidateResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&mutator.HelloMutator{},
			},
		},
		{
			name:   "validate job appends warnings",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(&api.Job{}, nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),

			wantProxyResponse: &api.JobValidateResponse{
				Warnings: helper.MergeMultierrorWarnings(errors.New("some warning")),
			},

			nomadResponse: toJson(t, &api.JobValidateResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		//Validate
		// {
		// 	name:        "validate job adds warnings",
		// 	path:        "/v1/validate/job",
		// 	method:      "POST",
		// 	requestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),

		// 	wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),
		// 	wantProxyResponseJson: toJson(t, &api.JobValidateResponse{

		// 		Warnings: "1 error occurred:\n\t* some warning\n\n",
		// 	}),

		// 	nomadResponse: toJson(t, &api.JobValidateResponse{}),
		// 	validators: []admissionctrl.JobValidator{
		// 		mockValidatorReturningWarnings("some warning"),
		// 	},
		// 	mutators: []admissionctrl.JobMutator{},
		// },
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadBackendCalled := false
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				nomadBackendCalled = true
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
			nomadClient := buildNomadClient(t, proxyServer)

			resp, _, err := tc.requestSender(nomadClient)

			assert.NoError(t, err, "No http call error")
			assert.Equal(t, tc.wantProxyResponse, resp, "OK response is expected")

			assert.True(t, nomadBackendCalled, "Nomad backend was called")

		})
	}

}
func TestJobUpdateProxy(t *testing.T) {

	type test struct {
		name        string
		path        string
		method      string
		requestJson string

		wantNomadRequestJson  string
		wantProxyResponseJson string
		nomadResponse         string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator
	}

	tests := []test{

		{
			name:        "update job adds hello meta (does this method actually exist?)",
			path:        "/v1/job/example",
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadBackendCalled := false
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				nomadBackendCalled = true
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
			assert.True(t, nomadBackendCalled, "Nomad backend was called")

		})
	}

}

func buildNomadClient(t *testing.T, proxyServer *httptest.Server) *api.Client {
	t.Helper()
	nomadClient, err := api.NewClient(&api.Config{
		Address: proxyServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return nomadClient
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
	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)

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
