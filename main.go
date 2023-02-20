package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/mxab/nacp/config"
)

func NewProxyHandler(nomadAddress *url.URL, jobHandler *admissionctrl.JobHandler, appLogger hclog.Logger) func(http.ResponseWriter, *http.Request) {

	// create a reverse proxy that catches "/v1/jobs" post calls
	// and forwards them to the jobs service
	// create a new reverse proxy

	proxy := httputil.NewSingleHostReverseProxy(nomadAddress)

	originalDirector := proxy.Director

	proxy.Director = func(r *http.Request) {
		originalDirector(r)
	}

	return func(w http.ResponseWriter, r *http.Request) {

		appLogger.Info("Request received", "path", r.URL.Path, "method", r.Method)

		// // only check for the header if it is a POST request and path is /v1/jobs
		if isCreate(r) || isUpdate(r) {

			jobRegisterRequest := &structs.JobRegisterRequest{}

			if err := json.NewDecoder(r.Body).Decode(jobRegisterRequest); err != nil {
				//TODO: configure if we want to prevent the request from being forwarded
				appLogger.Info("Failed decoding job, skipping admission controller", "error", err)
				return
			}
			job, warnings, err := jobHandler.ApplyAdmissionControllers(jobRegisterRequest.Job)
			if err != nil {

				appLogger.Warn("Admission controllers send an error, returning erro", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
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
			appLogger.Info("Job after admission controllers", "job", string(data))
			r.ContentLength = int64(len(data))
			r.Body = io.NopCloser(bytes.NewBuffer(data))

		}
		proxy.ServeHTTP(w, r)
	}

}
func isCreate(r *http.Request) bool {
	return r.Method == "PUT" && r.URL.Path == "/v1/jobs"
}
func isUpdate(r *http.Request) bool {
	return r.Method == "PUT" && regexp.MustCompile("^/v1/job/.*").MatchString(r.URL.Path)
}

// https://www.codedodle.com/go-reverse-proxy-example.html
// https://joshsoftware.wordpress.com/2021/05/25/simple-and-powerful-reverseproxy-in-go/
func main() {

	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "nacp",
		Level:  hclog.LevelFromString("DEBUG"),
		Output: os.Stdout,
	})

	appLogger.Info("Starting Nomad Admission Control Proxy")

	// and forwards them to the jobs service
	// create a new reverse proxy

	c, err := config.LoadConfig("config.hcl")
	if err != nil {
		appLogger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	backend, err := url.Parse(c.Nomad.Address)
	if err != nil {
		panic(err)
	}
	handler := admissionctrl.NewJobHandler(

		[]admissionctrl.JobMutator{&mutator.HelloMutator{}},
		[]admissionctrl.JobValidator{},
		appLogger.Named("handler"),
	)

	proxy := NewProxyHandler(backend, handler, appLogger)

	http.HandleFunc("/", proxy)

	appLogger.Info("Started Nomad Admission Control Proxy", "bind", c.Bind, "port", c.Port)
	appLogger.Error("%s", http.ListenAndServe(fmt.Sprintf("%s:%d", c.Bind, c.Port), nil))
}
