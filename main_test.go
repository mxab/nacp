package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxy(t *testing.T) {

	nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters

		assert.Equal(t, req.Method, "POST")
		assert.Equal(t, req.URL.String(), "/v1/jobs")

		x := req.Header.Get("Some-Header")
		assert.Equal(t, x, "Some Value", "Some-Header header has value 'Some Value'")
		// Send response to be tested
		rw.Write([]byte(`OK`))
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

	res, err := http.Post(fmt.Sprintf("%s/v1/jobs", proxyServer.URL), "application/json", strings.NewReader(`
	{}
	`))

	assert.NoError(t, err, "No http call error")
	assert.Equal(t, res.StatusCode, 200, "OK response is expected")
	// NACP header has value "hello"

}
