package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	tests := []test{
		{
			name:   "create job",
			path:   "/v1/jobs",
			method: "POST",
			want:   "CREATE",
		},
		{
			name:   "update job",
			path:   "/v1/job/some-job",
			method: "POST",
			want:   "UPDATE",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters

				assert.Equal(t, req.Method, tc.method, "Ensure method is set to POST")
				assert.Equal(t, req.URL.Path, tc.path, "Ensure path is set")
				assert.Equal(t, req.Header.Get("NACP"), tc.want, "Ensure NACP header is set to CREATE")

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
			proxy := NewServer(c)

			//http.Handle("/", proxy)

			proxyServer := httptest.NewServer(proxy)
			defer proxyServer.Close()

			res, err := http.Post(fmt.Sprintf("%s%s", proxyServer.URL, tc.path), "application/json", strings.NewReader(`
			{}
			`))
			assert.NoError(t, err, "No http call error")
			assert.Equal(t, res.StatusCode, 200, "OK response is expected")
		})
	}

}
