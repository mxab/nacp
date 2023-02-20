package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/go-hclog"
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
		name   string
		path   string
		method string
		want   string
	}
	wantJob := testutil.ReadJob(t, "job.json")
	wantJob.Meta = map[string]string{
		"hello": "world",
	}
	register := &api.JobRegisterRequest{
		Job: wantJob,
	}
	wantRegisterData, err := json.Marshal(register)
	if err != nil {
		t.Fatal(err)
	}
	tests := []test{
		{
			name:   "create job",
			path:   "/v1/jobs",
			method: "PUT",
			want:   string(wantRegisterData),
		},
		{
			name:   "update job",
			path:   "/v1/job/some-job",
			method: "PUT",
			want:   string(wantRegisterData),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters

				assert.Equal(t, req.Method, tc.method, "Ensure method is set to POST")
				assert.Equal(t, req.URL.Path, tc.path, "Ensure path is set")
				jsonData, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				json := string(jsonData)
				assert.JSONEq(t, tc.want, json, "Body matches")
				_, _ = rw.Write([]byte(`OK`))
			}))
			// Close the server when test finishes
			defer nomadDummy.Close()

			// Use Client & URL from our local test server

			nomad, err := url.Parse(nomadDummy.URL)
			if err != nil {
				t.Fatal(err)
			}
			jobHandler := admissionctrl.NewJobHandler(
				[]admissionctrl.JobMutator{
					&mutator.HelloMutator{},
				},
				[]admissionctrl.JobValidator{},
				hclog.NewNullLogger(),
			)
			proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger())

			proxyServer := httptest.NewServer(http.HandlerFunc(proxy))
			defer proxyServer.Close()

			jobRequest := api.JobRegisterRequest{
				Job: testutil.ReadJob(t, "job.json"),
			}

			buffer := new(bytes.Buffer)
			err = json.NewEncoder(buffer).Encode(jobRequest)
			if err != nil {
				t.Fatal(err)
			}

			res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, tc.path), buffer)
			assert.NoError(t, err, "No http call error")
			assert.Equal(t, 200, res.StatusCode, "OK response is expected")
		})
	}

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

	jobRequest := api.JobRegisterRequest{
		Job: testutil.ReadJob(t, "job.json"),
	}

	buffer := new(bytes.Buffer)
	err = json.NewEncoder(buffer).Encode(jobRequest)
	if err != nil {
		t.Fatal(err)
	}
	res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, "/v1/jobs"), buffer)
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
