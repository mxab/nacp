package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
)

// rewrite the test above as table driven test
func TestProxyTableDriven(t *testing.T) {

	type test struct {
		name   string
		path   string
		method string
		want   string
	}
	wantJob := testutil.ReadJob(t)
	wantJob.Meta = map[string]string{
		"hello": "world",
	}
	register := &structs.JobRegisterRequest{
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
			method: "POST",
			want:   string(wantRegisterData),
		},
		{
			name:   "update job",
			path:   "/v1/job/some-job",
			method: "POST",
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
			c := Config{
				Port: 8080,
				Bind: "0.0.0.0",
				nomad: NomadServer{
					Address: nomadDummy.URL,
				},
			}
			proxy := NewServer(c, hclog.NewNullLogger())

			//http.Handle("/", proxy)

			proxyServer := httptest.NewServer(proxy)
			defer proxyServer.Close()

			jobRequest := structs.JobRegisterRequest{
				Job: testutil.ReadJob(t),
			}

			buffer := new(bytes.Buffer)
			err := json.NewEncoder(buffer).Encode(jobRequest)
			if err != nil {
				t.Fatal(err)
			}
			res, err := http.Post(fmt.Sprintf("%s%s", proxyServer.URL, tc.path), "application/json", buffer)
			assert.NoError(t, err, "No http call error")
			assert.Equal(t, res.StatusCode, 200, "OK response is expected")
		})
	}

}
