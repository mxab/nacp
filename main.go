package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
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
		r.Header.Set("Some-Header", "Some Value")
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
