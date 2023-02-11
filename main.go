package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
)

type NomadServer struct {
	Address string
}
type Config struct {
	Port  int
	Bind  string
	nomad NomadServer
}

func NewServer(c Config, appLogger hclog.Logger) *httputil.ReverseProxy {

	// create a reverse proxy that catches "/v1/jobs" post calls
	// and forwards them to the jobs service
	// create a new reverse proxy
	backend, err := url.Parse(c.nomad.Address)
	if err != nil {
		panic(err)
	}

	handler := admissionctrl.NewJobHandler(

		[]admissionctrl.JobMutator{&mutator.HelloMutator{}},
		[]admissionctrl.JobValidator{},
		appLogger.Named("handler"),
	)

	proxy := httputil.NewSingleHostReverseProxy(backend)
	originalDirector := proxy.Director

	proxy.Director = func(r *http.Request) {
		originalDirector(r)

		// only check for the header if it is a POST request and path is /v1/jobs
		if isCreate(r) || isUpdate(r) {

			jobRegisterRequest := &structs.JobRegisterRequest{}
			if err := json.NewDecoder(r.Body).Decode(jobRegisterRequest); err != nil {
				//TODO: configure if we want to prevent the request from being forwarded
				appLogger.Info("Failed decoding job, skipping admission controller", "error", err)
				return
			}
			job, warnings, err := handler.ApplyAdmissionControllers(jobRegisterRequest.Job)
			if err != nil {
				//TODO: configure if we want to prevent the request from being forwarded

				appLogger.Warn("Failed to apply admission controllers, skipping", "error", err)
				return
			}
			if len(warnings) > 0 {
				appLogger.Warn("Warnings applying admission controllers", "warnings", warnings)
			}
			jobRegisterRequest.Job = job

			data, err := json.Marshal(jobRegisterRequest)

			if err != nil {
				//TODO: configure if we want to prevent the request from being forwarded
				appLogger.Warn("Error marshalling job", "error", err)
				return
			}
			r.ContentLength = int64(len(data))
			r.Body = io.NopCloser(bytes.NewBuffer(data))

		}

	}
	return proxy
}
func isCreate(r *http.Request) bool {
	return r.Method == "POST" && r.URL.Path == "/v1/jobs"
}
func isUpdate(r *http.Request) bool {
	return r.Method == "POST" && regexp.MustCompile("^/v1/job/.*").MatchString(r.URL.Path)
}

// https://www.codedodle.com/go-reverse-proxy-example.html
func main() {

	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "nacp",
		Level:  hclog.LevelFromString("DEBUG"),
		Output: os.Stdout,
	})

	appLogger.Info("Starting Nomad Admission Control Proxy")

	// and forwards them to the jobs service
	// create a new reverse proxy

	c := Config{
		Port: 8080,
		Bind: "0.0.0.0",
		nomad: NomadServer{
			Address: "http://localhost:4646",
		},
	}
	NewServer(c, appLogger)

}
