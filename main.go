package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
)

type NomadServer struct {
	Address string
}
type Config struct {
	Port  int
	Bind  string
	nomad NomadServer
}

func NewServer(c Config) *httputil.ReverseProxy {

	// create a reverse proxy that catches "/v1/jobs" post calls
	// and forwards them to the jobs service
	// create a new reverse proxy
	backend, err := url.Parse(c.nomad.Address)
	if err != nil {
		panic(err)
	}
	proxy := httputil.NewSingleHostReverseProxy(backend)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)

		// only check for the header if it is a POST request and path is /v1/jobs
		if r.Method == "POST" && r.URL.Path == "/v1/jobs" {
			r.Header.Set("NACP", "CREATE")
		}

		// only check for the header if it is a POST request
		// and path matches a regex that starts with /v1/job/ followed by a reg job name (e.g. /v1/job/some-job)
		if r.Method == "POST" && regexp.MustCompile("^/v1/job/.*").MatchString(r.URL.Path) {
			r.Header.Set("NACP", "UPDATE")
		}

	}
	return proxy
}

// https://www.codedodle.com/go-reverse-proxy-example.html
func main() {

	fmt.Println("Hello")
	// create a reverse proxy that catches "/v1/jobs" post calls
	// and forwards them to the jobs service
	// create a new reverse proxy

	c := Config{
		Port: 8080,
		Bind: "0.0.0.0",
		nomad: NomadServer{
			Address: "http://localhost:4646",
		},
	}
	NewServer(c)

}
