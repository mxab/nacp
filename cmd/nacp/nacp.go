package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/admissionctrl/validator"
	"github.com/mxab/nacp/config"
)

type contextKeyWarnings struct{}
type contextKeyValidationError struct{}

var (
	ctxWarnings        = contextKeyWarnings{}
	ctxValidationError = contextKeyValidationError{}
	jobPathRegex       = regexp.MustCompile(`^/v1/job/[a-zA-Z]+[a-z-Z0-9\-]*$`)
	jobPlanPathRegex   = regexp.MustCompile(`^/v1/job/[a-zA-Z]+[a-z-Z0-9\-]*/plan$`)
)

func NewProxyHandler(nomadAddress *url.URL, jobHandler *admissionctrl.JobHandler, appLogger hclog.Logger, transport *http.Transport) func(http.ResponseWriter, *http.Request) {

	proxy := httputil.NewSingleHostReverseProxy(nomadAddress)
	if transport != nil {
		proxy.Transport = transport
	}

	originalDirector := proxy.Director

	proxy.Director = func(r *http.Request) {
		originalDirector(r)
	}

	proxy.ModifyResponse = func(resp *http.Response) error {

		var err error

		if isRegister(resp.Request) {
			err = handRegisterResponse(resp, appLogger)
		} else if isPlan(resp.Request) {
			err = handleJobPlanResponse(resp, appLogger)
		} else if isValidate(resp.Request) {
			err = handleJobValdidateResponse(resp, appLogger)
		}
		if err != nil {
			appLogger.Error("Preparing response failed", "error", err)
			return err
		}

		return nil
	}

	return func(w http.ResponseWriter, r *http.Request) {

		appLogger.Info("Request received", "path", r.URL.Path, "method", r.Method)

		var err error
		//var err error
		if isRegister(r) {
			r, err = handleRegister(r, appLogger, jobHandler)

		} else if isPlan(r) {

			r, err = handlePlan(r, appLogger, jobHandler)

		} else if isValidate(r) {
			r, err = handleValidate(r, appLogger, jobHandler)

		}
		if err != nil {
			appLogger.Warn("Error applying admission controllers", "error", err)
			writeError(w, err)

		} else {
			proxy.ServeHTTP(w, r)
		}

	}

}

func handRegisterResponse(resp *http.Response, appLogger hclog.Logger) error {

	warnings, ok := resp.Request.Context().Value(ctxWarnings).([]error)
	if !ok && len(warnings) == 0 {
		return nil
	}

	response := &api.JobRegisterResponse{}
	reader := resp.Body

	isGzip, reader, err := checkIfGzipAndTransformReader(resp, reader)
	if err != nil {
		return err
	}
	defer reader.Close()
	if err := json.NewDecoder(reader).Decode(response); err != nil {
		return err
	}
	appLogger.Info("Job after admission controllers", "job", response.JobModifyIndex)

	response.Warnings = buildFullWarningMsg(response.Warnings, warnings)

	responeData, err := json.Marshal(response)

	if err != nil {
		return err
	}

	if isGzip {
		rewriteResponseGzip(resp, responeData)
	} else {
		rewriteResponse(resp, responeData)
	}

	return nil
}

func checkIfGzipAndTransformReader(resp *http.Response, reader io.ReadCloser) (bool, io.ReadCloser, error) {
	enc := resp.Header.Get("Content-Encoding")
	isGzip := enc == "gzip"
	if isGzip {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return false, nil, err
		}

		reader = gzipReader
	}
	return isGzip, reader, nil
}
func handleJobPlanResponse(resp *http.Response, appLogger hclog.Logger) error {
	warnings, ok := resp.Request.Context().Value(ctxWarnings).([]error)
	if !ok && len(warnings) == 0 {
		return nil
	}

	isGzip, reader, err := checkIfGzipAndTransformReader(resp, resp.Body)
	if err != nil {
		return err
	}
	defer reader.Close()

	response := &api.JobPlanResponse{}
	if err := json.NewDecoder(reader).Decode(response); err != nil {
		return err
	}
	appLogger.Info("Job after admission controllers", "job", response.JobModifyIndex)

	response.Warnings = buildFullWarningMsg(response.Warnings, warnings)

	responeData, err := json.Marshal(response)

	if err != nil {
		return err
	}

	if isGzip {
		rewriteResponseGzip(resp, responeData)
	} else {
		rewriteResponse(resp, responeData)
	}
	return nil
}
func handleJobValdidateResponse(resp *http.Response, appLogger hclog.Logger) error {

	ctx := resp.Request.Context()
	validationErr, okErr := ctx.Value(ctxValidationError).(error)
	warnings, okWarnings := resp.Request.Context().Value(ctxWarnings).([]error)
	if !okErr && !okWarnings {
		return nil
	}

	response := &api.JobValidateResponse{}
	isGzip, reader, err := checkIfGzipAndTransformReader(resp, resp.Body)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := json.NewDecoder(reader).Decode(response); err != nil {
		return err
	}

	if validationErr != nil {
		validationErrors := []string{}
		var validationError string
		if merr, ok := validationErr.(*multierror.Error); ok {
			for _, err := range merr.Errors {
				validationErrors = append(validationErrors, err.Error())
			}
			validationError = merr.Error()
		} else {
			validationErrors = append(validationErrors, validationErr.Error())
			validationError = err.Error()
		}

		response.ValidationErrors = validationErrors
		response.Error = validationError
	}

	if len(warnings) > 0 {
		response.Warnings = buildFullWarningMsg(response.Warnings, warnings)
	}

	responeData, err := json.Marshal(response)

	if err != nil {
		appLogger.Error("Error marshalling job", "error", err)
		return err
	}

	if isGzip {
		rewriteResponseGzip(resp, responeData)
	} else {
		rewriteResponse(resp, responeData)
	}

	return nil
}

func buildFullWarningMsg(upstreamResponseWarnings string, warnings []error) string {
	allWarnings := &multierror.Error{}

	if upstreamResponseWarnings != "" {
		multierror.Append(allWarnings, fmt.Errorf("%s", upstreamResponseWarnings))
	}
	allWarnings = multierror.Append(allWarnings, warnings...)
	warningMsg := helper.MergeMultierrorWarnings(allWarnings)
	return warningMsg
}

func rewriteResponse(resp *http.Response, newResponeData []byte) {
	resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newResponeData)))

	resp.ContentLength = int64(len(newResponeData))
	resp.Body = io.NopCloser(bytes.NewBuffer(newResponeData))
}
func rewriteResponseGzip(resp *http.Response, newResponeData []byte) {

	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	gz.Write(newResponeData)
	gz.Close()

	resp.Header.Set("Content-Length", strconv.Itoa(compressed.Len()))
	resp.ContentLength = int64(compressed.Len())

	resp.Body = io.NopCloser(&compressed)
}
func rewriteRequest(r *http.Request, data []byte) {

	r.ContentLength = int64(len(data))
	r.Body = io.NopCloser(bytes.NewBuffer(data))
}

func handleRegister(r *http.Request, appLogger hclog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {
	body := r.Body
	jobRegisterRequest := &api.JobRegisterRequest{}

	if err := json.NewDecoder(body).Decode(jobRegisterRequest); err != nil {

		return r, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobRegisterRequest.Job

	job, warnings, err := jobHandler.ApplyAdmissionControllers(orginalJob)
	if err != nil {
		return r, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}
	jobRegisterRequest.Job = job

	data, err := json.Marshal(jobRegisterRequest)

	if err != nil {
		return r, fmt.Errorf("error marshalling job: %w", err)
	}

	ctx := r.Context()
	if len(warnings) > 0 {
		ctx = context.WithValue(ctx, ctxWarnings, warnings)
	}

	appLogger.Info("Job after admission controllers", "job", string(data))
	r = r.WithContext(ctx)
	rewriteRequest(r, data)
	return r, nil
}
func handlePlan(r *http.Request, appLogger hclog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {
	body := r.Body
	jobPlanRequest := &api.JobPlanRequest{}

	if err := json.NewDecoder(body).Decode(jobPlanRequest); err != nil {
		return r, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobPlanRequest.Job

	job, warnings, err := jobHandler.ApplyAdmissionControllers(orginalJob)
	if err != nil {
		return r, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}

	jobPlanRequest.Job = job

	data, err := json.Marshal(jobPlanRequest)

	if err != nil {
		return r, fmt.Errorf("error marshalling job: %w", err)
	}
	ctx := r.Context()
	if len(warnings) > 0 {
		ctx = context.WithValue(ctx, ctxWarnings, warnings)

	}
	r = r.WithContext(ctx)
	appLogger.Info("Job after admission controllers", "job", string(data))
	rewriteRequest(r, data)
	return r, nil
}

func handleValidate(r *http.Request, appLogger hclog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {

	body := r.Body
	jobValidateRequest := &api.JobValidateRequest{}
	err := json.NewDecoder(body).Decode(jobValidateRequest)
	if err != nil {
		return r, err
	}
	job := jobValidateRequest.Job

	job, mutateWarnings, err := jobHandler.AdmissionMutators(job)

	if err != nil {
		return r, err
	}
	jobValidateRequest.Job = job

	validateWarnings, err := jobHandler.AdmissionValidators(job)
	//copied from https: //github.com/hashicorp/nomad/blob/v1.5.0/nomad/job_endpoint.go#L574

	ctx := r.Context()
	ctx = context.WithValue(ctx, ctxValidationError, err)

	validateWarnings = append(validateWarnings, mutateWarnings...)

	data, err := json.Marshal(jobValidateRequest)
	if err != nil {
		return r, err
	}

	if len(validateWarnings) > 0 {
		ctx = context.WithValue(ctx, ctxWarnings, validateWarnings)

	}
	r = r.WithContext(ctx)
	rewriteRequest(r, data)
	return r, nil

}

func writeError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}
func isRegister(r *http.Request) bool {
	isRegister := isCreate(r) || isUpdate(r)
	return isRegister
}

// cli does PUT, browser does POST :/
func isCreate(r *http.Request) bool {
	return (r.Method == "PUT" || r.Method == "POST") && r.URL.Path == "/v1/jobs"
}
func isUpdate(r *http.Request) bool {

	return (r.Method == "PUT" || r.Method == "POST") && jobPathRegex.MatchString(r.URL.Path)
}
func isPlan(r *http.Request) bool {

	return (r.Method == "PUT" || r.Method == "POST") && jobPlanPathRegex.MatchString(r.URL.Path)
}
func isValidate(r *http.Request) bool {

	return (r.Method == "PUT" || r.Method == "POST") && r.URL.Path == "/v1/validate/job"
}

// https://www.codedodle.com/go-reverse-proxy-example.html
// https://joshsoftware.wordpress.com/2021/05/25/simple-and-powerful-reverseproxy-in-go/
func main() {

	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:   "nacp",
		Level:  hclog.LevelFromString("DEBUG"),
		Output: os.Stdout,
	})

	c := buildConfig(appLogger)
	appLogger.SetLevel(hclog.LevelFromString(c.LogLevel))
	server, err := buildServer(c, appLogger)

	if err != nil {
		appLogger.Error("Failed to build server", "error", err)
		os.Exit(1)
	}

	var end error
	if c.Tls != nil {
		appLogger.Info("Starting NACP with TLS", "bind", c.Bind, "port", c.Port)
		end = server.ListenAndServeTLS(c.Tls.CertFile, c.Tls.KeyFile)
	} else {
		appLogger.Info("Starting NACP", "bind", c.Bind, "port", c.Port)
		end = server.ListenAndServe()
	}
	appLogger.Error("NACP stopped", "error", end)
}

func buildServer(c *config.Config, appLogger hclog.Logger) (*http.Server, error) {
	backend, err := url.Parse(c.Nomad.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nomad address: %w", err)

	}
	var transport *http.Transport
	if c.Nomad.TLS != nil {
		transport, err = buildCustomTransport(*c.Nomad.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom transport: %w", err)

		}
	}
	jobMutators, err := createMutators(c, appLogger.Named("mutators"))
	if err != nil {
		return nil, fmt.Errorf("failed to create mutators: %w", err)

	}
	jobValidators, err := createValidators(c, appLogger.Named("validators"))
	if err != nil {
		return nil, fmt.Errorf("failed to create validators: %w", err)

	}

	handler := admissionctrl.NewJobHandler(

		jobMutators,
		jobValidators,
		appLogger.Named("handler"),
	)

	proxy := NewProxyHandler(backend, handler, appLogger, transport)

	bind := fmt.Sprintf("%s:%d", c.Bind, c.Port)
	var tlsConfig *tls.Config

	if c.Tls != nil && c.Tls.CaFile != "" {
		tlsConfig, err = createTlsConfig(c.Tls.CaFile, c.Tls.NoClientCert)
		if err != nil {
			return nil, fmt.Errorf("failed to create tls config: %w", err)

		}
	}

	server := &http.Server{
		Addr:      bind,
		TLSConfig: tlsConfig,
		Handler:   http.HandlerFunc(proxy),
	}
	return server, nil
}

func buildConfig(logger hclog.Logger) *config.Config {

	configPtr := flag.String("config", "", "point to a nacp config file")
	flag.Parse()
	var c *config.Config

	if _, err := os.Stat(*configPtr); err == nil && *configPtr != "" {
		c, err = config.LoadConfig(*configPtr)
		if err != nil {
			logger.Error("Failed to load config", "error", err)
			os.Exit(1)
		}
		logger.Info("Loaded config", "config", *configPtr)
	} else {
		logger.Info("No config file found, using default config")
		c = config.DefaultConfig()
	}
	return c
}

func createTlsConfig(caFile string, noClientCert bool) (*tls.Config, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	clientAuth := tls.RequireAndVerifyClientCert
	if noClientCert {
		clientAuth = tls.NoClientCert
	}
	tlsConfig := &tls.Config{
		ClientCAs:  caCertPool,
		ClientAuth: clientAuth,
	}

	return tlsConfig, nil
}

func createMutators(c *config.Config, logger hclog.Logger) ([]admissionctrl.JobMutator, error) {
	var jobMutators []admissionctrl.JobMutator
	for _, m := range c.Mutators {
		switch m.Type {

		case "opa_json_patch":

			mutator, err := mutator.NewOpaJsonPatchMutator(m.Name, m.OpaRule.Filename, m.OpaRule.Query, logger.Named("opa_mutator"))
			if err != nil {
				return nil, err
			}
			jobMutators = append(jobMutators, mutator)

		case "json_patch_webhook":
			mutator, err := mutator.NewJsonPatchWebhookMutator(m.Name, m.Webhook.Endpoint, m.Webhook.Method, logger.Named("json_patch_webhook_mutator"))
			if err != nil {
				return nil, err
			}
			jobMutators = append(jobMutators, mutator)

		default:
			return nil, fmt.Errorf("unknown mutator type %s", m.Type)
		}

	}
	return jobMutators, nil
}
func createValidators(c *config.Config, logger hclog.Logger) ([]admissionctrl.JobValidator, error) {
	var jobValidators []admissionctrl.JobValidator
	for _, v := range c.Validators {
		switch v.Type {
		case "opa":

			opaValidator, err := validator.NewOpaValidator(v.Name, v.OpaRule.Filename, v.OpaRule.Query, logger.Named("opa_validator"))
			if err != nil {
				return nil, err
			}
			jobValidators = append(jobValidators, opaValidator)

		case "webhook":
			validator, err := validator.NewWebhookValidator(v.Name, v.Webhook.Endpoint, v.Webhook.Method, logger.Named("webhook_validator"))
			if err != nil {
				return nil, err
			}
			jobValidators = append(jobValidators, validator)
		default:
			return nil, fmt.Errorf("unknown validator type %s", v.Type)
		}

	}
	return jobValidators, nil
}

func buildCustomTransport(config config.NomadServerTLS) (*http.Transport, error) {
	// Create a custom transport to allow for self-signed certs
	// and to allow for a custom timeout

	//load key pair
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return nil, err
	}

	// create CA pool
	caCert, err := os.ReadFile(config.CaFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,

			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		},
	}
	return transport, err
}
