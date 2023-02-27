package main

import (
	"bytes"
	"context"
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
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/admissionctrl/opa"
	"github.com/mxab/nacp/admissionctrl/validator"
	"github.com/mxab/nacp/config"
)

type contextKeyWarnings struct{}

var (
	ctxWarnings      = contextKeyWarnings{}
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

	proxy.ModifyResponse = func(resp *http.Response) error {

		warnings, ok := resp.Request.Context().Value(ctxWarnings).([]error)
		if ok && len(warnings) > 0 {
			appLogger.Warn("Warnings applying admission controllers", "warnings", warnings)

			var response interface{}
			var warningsGetter func() string
			var warningsSetter func(string)
			var err error
			if isRegister(resp.Request) {
				response, warningsGetter, warningsSetter, err = getRegisterResponseParts(resp, appLogger)
			}
			if isPlan(resp.Request) {
				response, warningsGetter, warningsSetter, err = getJobPlanResponseParts(resp, appLogger)
			}
			if err != nil {
				appLogger.Error("Preparing response failed", "error", err)

				return err
			}
			newResponeData, err := appendWarnings(response, warnings, warningsGetter, warningsSetter)
			if err != nil {
				appLogger.Error("Error marshalling job", "error", err)
				return err
			}
			rewriteResponse(resp, newResponeData)

		}
		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {

		appLogger.Info("Request received", "path", r.URL.Path, "method", r.Method)

		ctx := r.Context()

		if isRegister(r) {
			data, warnings, err := handleRegister(r, appLogger, jobHandler)
			if err != nil {
				appLogger.Warn("Error applying admission controllers", "error", err)
				writeError(w, err)
				return
			}
			if len(warnings) > 0 {
				ctx = context.WithValue(ctx, ctxWarnings, warnings)
			}
			appLogger.Info("Job after admission controllers", "job", string(data))
			rewriteRequest(r, data)

		} else if isPlan(r) {

			data, warnings, err := handlePlan(r, appLogger, jobHandler)
			if err != nil {
				appLogger.Warn("Error applying admission controllers", "error", err)
				writeError(w, err)
				return
			}
			if len(warnings) > 0 {
				ctx = context.WithValue(ctx, ctxWarnings, warnings)

			}
			appLogger.Info("Job after admission controllers", "job", string(data))
			rewriteRequest(r, data)

		}
		r = r.WithContext(ctx)
		proxy.ServeHTTP(w, r)
	}

}

func getRegisterResponseParts(resp *http.Response, appLogger hclog.Logger) (interface{}, func() string, func(warnings string), error) {
	response := &api.JobRegisterResponse{}
	err := json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		appLogger.Error("Error decoding job", "error", err)
		return nil, nil, nil, err
	}
	appLogger.Info("Job after admission controllers", "job", response.JobModifyIndex)

	warningsGetter := func() string {
		return response.Warnings
	}
	warningsSetter := func(warnings string) {
		response.Warnings = warnings
	}
	return response, warningsGetter, warningsSetter, nil
}
func getJobPlanResponseParts(resp *http.Response, appLogger hclog.Logger) (interface{}, func() string, func(warnings string), error) {
	response := &api.JobPlanResponse{}
	err := json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		appLogger.Error("Error decoding job", "error", err)
		return nil, nil, nil, err
	}
	appLogger.Info("Job after admission controllers", "job", response.JobModifyIndex)

	warningsGetter := func() string {
		return response.Warnings
	}
	warningsSetter := func(warnings string) {
		response.Warnings = warnings
	}
	return response, warningsGetter, warningsSetter, nil
}

func appendWarnings(response interface{}, warnings []error, warningsGetter func() string, warningsSetter func(string)) ([]byte, error) {
	allWarnings := &multierror.Error{}

	upstreamResponseWarnings := warningsGetter()
	if upstreamResponseWarnings != "" {
		multierror.Append(allWarnings, fmt.Errorf("%s", upstreamResponseWarnings))
	}
	allWarnings = multierror.Append(allWarnings, warnings...)

	warningsSetter(allWarnings.Error())

	newResponeData, err := json.Marshal(response)
	if err != nil {

		return nil, err
	}

	return newResponeData, nil
}

func rewriteResponse(resp *http.Response, newResponeData []byte) {
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newResponeData)))
	resp.ContentLength = int64(len(newResponeData))
	resp.Body = io.NopCloser(bytes.NewBuffer(newResponeData))
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
		return nil, nil, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobPlanRequest.Job

	job, warnings, err := jobHandler.ApplyAdmissionControllers(orginalJob)
	if err != nil {
		return nil, nil, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}

	jobPlanRequest.Job = job

	data, err := json.Marshal(jobPlanRequest)

	if err != nil {
		return nil, nil, fmt.Errorf("error marshalling job: %w", err)
	}
	return data, warnings, nil
}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
func isRegister(r *http.Request) bool {
	isRegister := isCreate(r) || isUpdate(r)
	return isRegister
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
