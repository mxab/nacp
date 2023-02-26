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

var (
	jobPathRegex     = regexp.MustCompile(`^/v1/job/[a-zA-Z]+[a-z-Z0-9\-]*$`)
	jobPlanPathRegex = regexp.MustCompile(`^/v1/job/[a-zA-Z]+[a-z-Z0-9\-]*/plan$`)
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

		if isRegister {
			data, warnings, err := handleRegister(r, appLogger, jobHandler)
			if err != nil {
				writeError(w, err)
				return
			}
			if len(warnings) > 0 {
				//TODO: attach to response?
				appLogger.Warn("Warnings applying admission controllers", "warnings", warnings)
			}
			appLogger.Info("Job after admission controllers", "job", string(data))
			rewriteRequest(r, data)

		} else if isPlan(r) {

			data, warnings, err := handlePlan(r, appLogger, jobHandler)
			if err != nil {
				writeError(w, err)
				return
			}
			if len(warnings) > 0 {
				//TODO: attach to response?
				appLogger.Warn("Warnings applying admission controllers", "warnings", warnings)
			}
			appLogger.Info("Job after admission controllers", "job", string(data))
			rewriteRequest(r, data)

		}

		proxy.ServeHTTP(w, r)
	}

}

func rewriteRequest(r *http.Request, data []byte) {

	r.ContentLength = int64(len(data))
	r.Body = io.NopCloser(bytes.NewBuffer(data))
}

func handleRegister(r *http.Request, appLogger hclog.Logger, jobHandler *admissionctrl.JobHandler) ([]byte, []error, error) {
	body := r.Body
	jobRegisterRequest := &api.JobRegisterRequest{}

	if err := json.NewDecoder(body).Decode(jobRegisterRequest); err != nil {

		return nil, nil, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobRegisterRequest.Job

	job, warnings, err := jobHandler.ApplyAdmissionControllers(orginalJob)
	if err != nil {
		return nil, nil, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}
	jobRegisterRequest.Job = job

	data, err := json.Marshal(jobRegisterRequest)

	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling job: %w", err)
	}
	return data, warnings, nil
}
func handlePlan(r *http.Request, appLogger hclog.Logger, jobHandler *admissionctrl.JobHandler) ([]byte, []error, error) {
	body := r.Body
	jobPlanRequest := &api.JobPlanRequest{}

	if err := json.NewDecoder(body).Decode(jobPlanRequest); err != nil {
		appLogger.Info("Failed decoding job, skipping admission controller", "error", err)
		return nil, nil, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobPlanRequest.Job

	job, warnings, err := jobHandler.ApplyAdmissionControllers(orginalJob)
	if err != nil {
		appLogger.Warn("Admission controllers send an error, returning error", "error", err)
		return nil, nil, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}

	jobPlanRequest.Job = job

	data, err := json.Marshal(jobPlanRequest)

	if err != nil {
		appLogger.Warn("Error marshalling job", "error", err)
		return nil, nil, fmt.Errorf("error marshalling job: %w", err)
	}
	return data, warnings, nil
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
func isCreate(r *http.Request) bool {
	return r.Method == "PUT" && r.URL.Path == "/v1/jobs"
}
func isUpdate(r *http.Request) bool {

	return r.Method == "PUT" && jobPathRegex.MatchString(r.URL.Path)
}
func isPlan(r *http.Request) bool {

	return r.Method == "PUT" && jobPlanPathRegex.MatchString(r.URL.Path)
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
