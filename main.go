package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/admissionctrl/opa"
	"github.com/mxab/nacp/admissionctrl/validator"
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

		isRegister := isCreate(r) || isUpdate(r)

		//isPlan := r.Method == "POST" && r.URL.Path == "/v1/jobs/plan"
		if isRegister {

			jobRegisterRequest := &api.JobRegisterRequest{}

			if err := json.NewDecoder(r.Body).Decode(jobRegisterRequest); err != nil {
				appLogger.Info("Failed decoding job, skipping admission controller", "error", err)
				return
			}
			job, warnings, err := jobHandler.ApplyAdmissionControllers(jobRegisterRequest.Job)
			if err != nil {
				appLogger.Warn("Admission controllers send an error, returning erro", "error", err)
				writeError(w, err)
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

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
func isCreate(r *http.Request) bool {
	return r.Method == "PUT" && r.URL.Path == "/v1/jobs"
}
func isUpdate(r *http.Request) bool {

	return r.Method == "PUT" && regexp.MustCompile("^/v1/job/[a-zA-Z]+[a-z-Z0-9\\-]*$").MatchString(r.URL.Path)
}
func isPlan(r *http.Request) bool {
	return r.Method == "PUT" && regexp.MustCompile("^/v1/job/[a-zA-Z]+[a-z-Z0-9\\-]*/plan$").MatchString(r.URL.Path)
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
	configPtr := flag.String("config", "", "point to a nacp config file")
	flag.Parse()
	var c *config.Config

	if _, err := os.Stat(*configPtr); err == nil && *configPtr != "" {
		c, err = config.LoadConfig(*configPtr)
		if err != nil {
			appLogger.Error("Failed to load config", "error", err)
			os.Exit(1)
		}
	} else {
		c = config.DefaultConfig()
	}

	backend, err := url.Parse(c.Nomad.Address)
	if err != nil {
		appLogger.Error("Failed to parse nomad address", "error", err)
		os.Exit(1)
	}

	jobMutators, err := createMutatators(c, appLogger)
	if err != nil {
		appLogger.Error("Failed to create mutators", "error", err)
		os.Exit(1)
	}
	jobValidators, err := createValidators(c, appLogger)
	if err != nil {
		appLogger.Error("Failed to create validators", "error", err)
		os.Exit(1)
	}

	handler := admissionctrl.NewJobHandler(

		jobMutators,
		jobValidators,
		appLogger.Named("handler"),
	)

	proxy := NewProxyHandler(backend, handler, appLogger)

	http.HandleFunc("/", proxy)

	appLogger.Info("Started Nomad Admission Control Proxy", "bind", c.Bind, "port", c.Port)
	appLogger.Error("NACP stopped", "error", http.ListenAndServe(fmt.Sprintf("%s:%d", c.Bind, c.Port), nil))
}

func createMutatators(c *config.Config, appLogger hclog.Logger) ([]admissionctrl.JobMutator, error) {
	var jobMutators []admissionctrl.JobMutator
	for _, m := range c.Mutators {
		switch m.Type {
		case "hello":
			jobMutators = append(jobMutators, &mutator.HelloMutator{})

		case "opa_jsonpatch":

			opaRules := []opa.OpaQueryAndModule{}
			for _, r := range m.OpaRules {
				opaRules = append(opaRules, opa.OpaQueryAndModule{
					Filename: r.Filename,
					Query:    r.Query,
				})
			}
			mutator, err := mutator.NewOpaJsonPatchMutator(opaRules, appLogger.Named("opa_mutator"))
			if err != nil {
				return nil, err
			}
			jobMutators = append(jobMutators, mutator, mutator)

		}

	}
	return jobMutators, nil
}
func createValidators(c *config.Config, appLogger hclog.Logger) ([]admissionctrl.JobValidator, error) {
	var jobValidators []admissionctrl.JobValidator
	for _, v := range c.Validators {
		switch v.Type {
		case "opa":

			opaRules := []opa.OpaQueryAndModule{}
			for _, r := range v.OpaRules {
				opaRules = append(opaRules, opa.OpaQueryAndModule{
					Filename: r.Filename,
					Query:    r.Query,
				})
			}
			opaValidator, err := validator.NewOpaValidator(opaRules, appLogger.Named("opa_validator"))
			if err != nil {
				return nil, err
			}
			jobValidators = append(jobValidators, opaValidator)

		}
	}
	return jobValidators, nil
}
