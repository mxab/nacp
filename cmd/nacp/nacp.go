package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mxab/nacp/pkg/admissionctrl/remoteutil"
	"github.com/mxab/nacp/pkg/admissionctrl/types"
	"github.com/mxab/nacp/pkg/logutil"
	nacpOtel "github.com/mxab/nacp/pkg/otel"
	"github.com/open-policy-agent/opa/v1/sdk"

	"log/slog"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/nomad/api"
	"github.com/mxab/nacp/pkg/admissionctrl"
	"github.com/mxab/nacp/pkg/admissionctrl/mutator"
	"github.com/mxab/nacp/pkg/admissionctrl/notation"
	"github.com/mxab/nacp/pkg/admissionctrl/validator"
	"github.com/mxab/nacp/pkg/config"
	"github.com/mxab/nacp/pkg/helper"
	"github.com/notaryproject/notation-go/dir"
	"github.com/notaryproject/notation-go/verifier/truststore"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	version = "dev"
)

type contextKeyWarnings struct{}
type contextKeyValidationError struct{}
type contextKeyRequestContext struct{}

var (
	ctxWarnings        = contextKeyWarnings{}
	ctxValidationError = contextKeyValidationError{}
	ctxRequestContext  = contextKeyRequestContext{}
	jobPathRegex       = regexp.MustCompile(`^/v1/job/[^/]+$`)
	jobPlanPathRegex   = regexp.MustCompile(`^/v1/job/[^/]+/plan$`)

	nomadTimeout         = 310 * time.Second
	tokenResolveTimeout  = 30 * time.Second
	shutdownTimeout      = 30 * time.Second
	maxAdmissionBodySize = int64(32 << 20)
)

// New function to get client IP
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func resolveTokenAccessor(ctx context.Context, transport http.RoundTripper, nomadAddress *url.URL, token string) (*api.ACLToken, error) {
	if token == "" {
		return nil, nil
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   tokenResolveTimeout,
	}
	if transport == nil {
		client.Transport = http.DefaultTransport
	}

	selfURL := *nomadAddress
	selfURL.Path = "/v1/acl/token/self"

	req, err := http.NewRequestWithContext(ctx, "GET", selfURL.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Nomad-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to resolve token: %s", resp.Status)
	}

	var aclToken api.ACLToken
	if err := json.NewDecoder(resp.Body).Decode(&aclToken); err != nil {
		return nil, err
	}

	return &aclToken, nil
}
func NewProxyAsHandlerFunc(nomadAddress *url.URL, jobHandler *admissionctrl.JobHandler, logger *slog.Logger, transport http.RoundTripper) http.HandlerFunc {

	proxy := newProxyHandler(nomadAddress, jobHandler, logger, transport)
	handlerFunc := http.HandlerFunc(proxy)
	handlerFunc = otelhttp.NewHandler(handlerFunc, "/").(http.HandlerFunc)

	return handlerFunc
}
func newProxyHandler(nomadAddress *url.URL, jobHandler *admissionctrl.JobHandler, logger *slog.Logger, transport http.RoundTripper) func(http.ResponseWriter, *http.Request) {

	proxy := httputil.NewSingleHostReverseProxy(nomadAddress)

	if transport != nil {
		proxy.Transport = transport
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		return modifyProxyResponse(resp, logger)
	}
	var proxyHandler http.Handler = proxy

	nacpHandler := func(w http.ResponseWriter, r *http.Request) {

		r, err := resolveRequestContext(w, r, jobHandler, nomadAddress, transport, logger)
		if err != nil {
			logger.ErrorContext(r.Context(), "Resolving token failed", "error", err)
			writeError(w, err)
			return
		}

		r, err = applyAdmission(r, logger, jobHandler)
		if err != nil {
			logger.WarnContext(r.Context(), "Error applying admission controllers", "error", err)
			writeError(w, err)
			return
		}

		proxyHandler.ServeHTTP(w, r)
	}
	return nacpHandler

}

func modifyProxyResponse(resp *http.Response, logger *slog.Logger) error {

	var err error

	if isRegister(resp.Request) {
		err = handRegisterResponse(resp, logger)
	} else if isPlan(resp.Request) {
		err = handleJobPlanResponse(resp, logger)
	} else if isValidate(resp.Request) {
		err = handleJobValdidateResponse(resp, logger)
	}
	if err != nil {
		logger.ErrorContext(resp.Request.Context(), "Preparing response failed", "error", err)
		return err
	}

	return nil
}

// resolveRequestContext attaches the NACP request context to r, resolving the
// Nomad token first if any admission controller asked for it. The returned
// request always carries a usable context, even when resolving fails.
func resolveRequestContext(w http.ResponseWriter, r *http.Request, jobHandler *admissionctrl.JobHandler, nomadAddress *url.URL, transport http.RoundTripper, logger *slog.Logger) (*http.Request, error) {

	ctx := r.Context()
	reqCtx := &config.RequestContext{
		ClientIP:     getClientIP(r),
		ResolveToken: jobHandler.ResolveToken(),
	}

	isAdmissionActionable := isRegister(r) || isPlan(r) || isValidate(r)
	if isAdmissionActionable {
		r.Body = http.MaxBytesReader(w, r.Body, maxAdmissionBodySize)
	}
	if isAdmissionActionable && jobHandler.ResolveToken() {
		token := r.Header.Get("X-Nomad-Token")
		tokenInfo, err := resolveTokenAccessor(ctx, transport, nomadAddress, token)
		if err != nil {
			return r, err
		}
		if tokenInfo != nil {
			reqCtx.AccessorID = tokenInfo.AccessorID
			reqCtx.TokenInfo = config.SanitizeACLToken(tokenInfo)
		}
		logger.InfoContext(ctx, "Request received", "path", r.URL.Path, "method", r.Method, "clientIP", reqCtx.ClientIP, "accessorID", reqCtx.AccessorID)
	} else {
		logger.InfoContext(ctx, "Request received", "path", r.URL.Path, "method", r.Method, "clientIP", reqCtx.ClientIP)
	}

	return r.WithContext(context.WithValue(ctx, ctxRequestContext, reqCtx)), nil
}

func applyAdmission(r *http.Request, logger *slog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {
	if isRegister(r) {
		return handleRegister(r, logger, jobHandler)
	}
	if isPlan(r) {
		return handlePlan(r, logger, jobHandler)
	}
	if isValidate(r) {
		return handleValidate(r, logger, jobHandler)
	}
	return r, nil
}

func handRegisterResponse(resp *http.Response, logger *slog.Logger) error {

	warnings, ok := resp.Request.Context().Value(ctxWarnings).([]error)
	if !ok || len(warnings) == 0 || !isSuccessfulResponse(resp) {
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

	response.Warnings = buildFullWarningMsg(response.Warnings, warnings)

	responseData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	if isGzip {
		return rewriteResponseGzip(resp, responseData)
	}
	rewriteResponse(resp, responseData)
	return nil
}

type gzipResponseReader struct {
	*gzip.Reader
	source io.Closer
}

func (r *gzipResponseReader) Close() error {
	return errors.Join(r.Reader.Close(), r.source.Close())
}

func checkIfGzipAndTransformReader(resp *http.Response, reader io.ReadCloser) (bool, io.ReadCloser, error) {
	isGzip := resp.Header.Get("Content-Encoding") == "gzip"
	if !isGzip {
		return false, reader, nil
	}
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		reader.Close()
		return false, nil, err
	}
	return true, &gzipResponseReader{Reader: gzipReader, source: reader}, nil
}

func isSuccessfulResponse(resp *http.Response) bool {
	return resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices
}
func handleJobPlanResponse(resp *http.Response, logger *slog.Logger) error {
	warnings, ok := resp.Request.Context().Value(ctxWarnings).([]error)
	if !ok || len(warnings) == 0 || !isSuccessfulResponse(resp) {
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

	response.Warnings = buildFullWarningMsg(response.Warnings, warnings)

	responseData, err := json.Marshal(response)
	if err != nil {
		return err
	}

	if isGzip {
		return rewriteResponseGzip(resp, responseData)
	}
	rewriteResponse(resp, responseData)
	return nil
}
func handleJobValdidateResponse(resp *http.Response, logger *slog.Logger) error {

	ctx := resp.Request.Context()
	validationErr, okErr := ctx.Value(ctxValidationError).(error)
	warnings, okWarnings := resp.Request.Context().Value(ctxWarnings).([]error)
	if (!okErr && !okWarnings) || !isSuccessfulResponse(resp) {
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
			validationError = validationErr.Error()
		}

		response.ValidationErrors = validationErrors
		response.Error = validationError
	}

	if len(warnings) > 0 {
		response.Warnings = buildFullWarningMsg(response.Warnings, warnings)
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("error marshalling job: %w", err)
	}

	if isGzip {
		return rewriteResponseGzip(resp, responseData)
	}
	rewriteResponse(resp, responseData)
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
func rewriteResponseGzip(resp *http.Response, newResponseData []byte) error {
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	if _, err := gz.Write(newResponseData); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}

	resp.Header.Set("Content-Length", strconv.Itoa(compressed.Len()))
	resp.ContentLength = int64(compressed.Len())
	resp.Body = io.NopCloser(&compressed)
	return nil
}
func rewriteRequest(r *http.Request, data []byte) {

	r.ContentLength = int64(len(data))
	r.Body = io.NopCloser(bytes.NewBuffer(data))
}

func handleRegister(r *http.Request, appLogger *slog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {
	body := r.Body
	jobRegisterRequest := &api.JobRegisterRequest{}

	ctx := r.Context()
	if err := json.NewDecoder(body).Decode(jobRegisterRequest); err != nil {

		return r, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobRegisterRequest.Job
	payload := &types.Payload{
		Job: orginalJob,
	}

	if reqCtx, ok := ctx.Value(ctxRequestContext).(*config.RequestContext); ok {
		payload.Context = reqCtx
	}

	job, warnings, err := jobHandler.ApplyAdmissionControllers(ctx, payload)
	if err != nil {
		return r, fmt.Errorf("admission controllers send an error, returning error: %w", err)
	}
	jobRegisterRequest.Job = job

	data, err := json.Marshal(jobRegisterRequest)

	if err != nil {
		return r, fmt.Errorf("error marshalling job: %w", err)
	}

	if len(warnings) > 0 {
		ctx = context.WithValue(ctx, ctxWarnings, warnings)
	}

	appLogger.Debug("Job after admission controllers", "job", string(data))
	r = r.WithContext(ctx)
	rewriteRequest(r, data)
	return r, nil
}
func handlePlan(r *http.Request, appLogger *slog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {
	body := r.Body
	jobPlanRequest := &api.JobPlanRequest{}

	if err := json.NewDecoder(body).Decode(jobPlanRequest); err != nil {
		return r, fmt.Errorf("failed decoding job, skipping admission controller: %w", err)
	}
	orginalJob := jobPlanRequest.Job
	payload := &types.Payload{
		Job: orginalJob,
	}

	if reqCtx, ok := r.Context().Value(ctxRequestContext).(*config.RequestContext); ok {
		payload.Context = reqCtx
	}

	job, warnings, err := jobHandler.ApplyAdmissionControllers(r.Context(), payload)
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
	appLogger.Debug("Job after admission controllers", "job", string(data))
	rewriteRequest(r, data)
	return r, nil
}

func handleValidate(r *http.Request, applogger *slog.Logger, jobHandler *admissionctrl.JobHandler) (*http.Request, error) {

	ctx := r.Context()
	body := r.Body
	jobValidateRequest := &api.JobValidateRequest{}
	err := json.NewDecoder(body).Decode(jobValidateRequest)
	if err != nil {
		return r, err
	}
	job := jobValidateRequest.Job
	payload := &types.Payload{
		Job: job,
	}

	if reqCtx, ok := ctx.Value(ctxRequestContext).(*config.RequestContext); ok {
		payload.Context = reqCtx
	}

	job, mutateWarnings, err := jobHandler.AdmissionMutators(ctx, payload)
	if err != nil {
		return r, err
	}
	jobValidateRequest.Job = job
	payload.Job = job

	validateWarnings, err := jobHandler.AdmissionValidators(ctx, payload)
	//copied from https: //github.com/hashicorp/nomad/blob/v1.5.0/nomad/job_endpoint.go#L574

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

func buildSlogHandler(json bool, level slog.Level) slog.Handler {
	opts := &slog.HandlerOptions{
		Level: level,
	}
	if json {
		return slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.NewTextHandler(os.Stdout, opts)
}

// https://www.codedodle.com/go-reverse-proxy-example.html
// https://joshsoftware.wordpress.com/2021/05/25/simple-and-powerful-reverseproxy-in-go/

func main() {

	configPtr := flag.String("config", "", "point to a nacp config file")
	bootstrapLoggerHandlerPtr := flag.Bool("bootstrap-json-logger", false, "use json for initial logging until config is loaded")
	flag.Parse()

	slog.SetDefault(slog.New(buildSlogHandler(*bootstrapLoggerHandlerPtr, slog.LevelInfo)))

	c, err := buildConfig(*configPtr)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	if err := run(c); err != nil {

		slog.Error("Error running nacp", "error", err)
		os.Exit(1)
	}
}

func run(c *config.Config) (err error) {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rootFactory, leveler := logutil.NewLoggerFactoryFromConfig(c.Telemetry.Logging)

	appLogger := rootFactory.GetLogger("nacp")
	slog.SetDefault(appLogger)

	setupOtel := *c.Telemetry.Logging.OtelLogging.Enabled || c.Telemetry.Metrics.Enabled || c.Telemetry.Tracing.Enabled
	if setupOtel {
		// Set up OpenTelemetry.
		otelShutdown, err := nacpOtel.SetupOTelSDK(ctx, *c.Telemetry.Logging.OtelLogging.Enabled, c.Telemetry.Metrics.Enabled, c.Telemetry.Tracing.Enabled, version, leveler.GetSeverietier())
		if err != nil {
			return fmt.Errorf("failed to setup OpenTelemetry: %w", err)
		}
		// Handle shutdown properly so nothing leaks.
		// https://opentelemetry.io/docs/languages/go/getting-started/
		defer func() {
			err = errors.Join(err, otelShutdown(context.Background()))
		}()

	}

	opaSDK, stopOPA, err := setupOpaSDK(ctx, appLogger, c.OpaSdk)
	if err != nil {
		return err
	}
	if stopOPA != nil {
		defer stopOPA()
	}

	server, err := buildServer(c, rootFactory, opaSDK)

	if err != nil {
		return fmt.Errorf("failed to build server: %w", err)
	}

	srvErr := make(chan error, 1)

	go func() {

		if c.Tls != nil {

			appLogger.Info("Starting NACP with TLS", "bind", c.Bind, "port", c.Port)
			srvErr <- server.ListenAndServeTLS(c.Tls.CertFile, c.Tls.KeyFile)
		} else {
			appLogger.Info("Starting NACP", "bind", c.Bind, "port", c.Port)
			srvErr <- server.ListenAndServe()
		}
	}()
	select {
	case err = <-srvErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		appLogger.Error("NACP stopped", "error", err)
		return err
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}
	appLogger.Info("Received shutdown signal, shutting down NACP")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shut down server: %w", err)
	}
	appLogger.Info("NACP stopped")
	return nil

}

func buildServer(c *config.Config, loggerFactory *logutil.LoggerFactory, sdk *sdk.OPA) (*http.Server, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	backend, err := url.Parse(c.Nomad.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nomad address: %w", err)
	}
	proxyTransport := http.DefaultTransport.(*http.Transport).Clone()
	proxyTransport.DialContext = (&net.Dialer{
		Timeout:   nomadTimeout,
		KeepAlive: nomadTimeout,
	}).DialContext
	proxyTransport.TLSHandshakeTimeout = nomadTimeout

	instrumentedProxyTransport := remoteutil.InstrumentedTransport(proxyTransport)

	if c.Nomad.TLS != nil {
		nomadTlsConfig, err := buildTlsConfig(*c.Nomad.TLS)
		if err != nil {
			return nil, fmt.Errorf("failed to create custom transport: %w", err)

		}
		proxyTransport.TLSClientConfig = nomadTlsConfig
	}

	jobMutators, resolveTokenMutators, err := createMutators(c, loggerFactory, sdk)
	if err != nil {
		return nil, fmt.Errorf("failed to create mutators: %w", err)
	}

	jobValidators, resolveTokenValidators, err := createValidators(c, loggerFactory, sdk)
	if err != nil {
		return nil, fmt.Errorf("failed to create validators: %w", err)
	}

	var resolveToken bool
	if resolveTokenMutators || resolveTokenValidators {
		resolveToken = true
	}

	jobHandler := admissionctrl.NewJobHandler(

		jobMutators,
		jobValidators,
		loggerFactory.GetLogger("handler"),
		resolveToken,
	)

	handlerFunc := NewProxyAsHandlerFunc(backend, jobHandler, loggerFactory.GetLogger("proxy-handler"), instrumentedProxyTransport)

	bind := fmt.Sprintf("%s:%d", c.Bind, c.Port)
	var tlsConfig *tls.Config

	if c.Tls != nil {
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		if c.Tls.CaFile != "" {
			tlsConfig, err = createTlsConfig(c.Tls.CaFile, c.Tls.NoClientCert)
			if err != nil {
				return nil, fmt.Errorf("failed to create tls config: %w", err)
			}
		}
	}

	server := &http.Server{
		Addr:              bind,
		TLSConfig:         tlsConfig,
		Handler:           handlerFunc,
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       nomadTimeout,
		WriteTimeout:      nomadTimeout,
		IdleTimeout:       120 * time.Second,
	}
	return server, nil
}

func buildConfig(configPath string) (*config.Config, error) {
	if configPath != "" {
		c, err := config.LoadConfig(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		slog.Info("Loaded config", "config", configPath)
		return c, nil
	}

	dir, _ := os.Getwd()
	slog.Info("No config file configured, using default config", "working_dir", dir)
	c := config.DefaultConfig()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid default config: %w", err)
	}
	return c, nil
}

func createTlsConfig(caFile string, noClientCert bool) (*tls.Config, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("CA file %q does not contain a valid certificate", caFile)
	}
	clientAuth := tls.RequireAndVerifyClientCert
	if noClientCert {
		clientAuth = tls.NoClientCert
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientCAs:  caCertPool,
		ClientAuth: clientAuth,
	}

	return tlsConfig, nil
}

func createMutators(c *config.Config, loggerFactory *logutil.LoggerFactory, opaSDK *sdk.OPA) ([]admissionctrl.JobMutator, bool, error) {
	jobMutators := make([]admissionctrl.JobMutator, 0, len(c.Mutators))
	var resolveToken bool
	for _, mutatorConfig := range c.Mutators {
		resolveToken = resolveToken || mutatorConfig.ResolveToken
		jobMutator, err := createMutator(mutatorConfig, loggerFactory, opaSDK)
		if err != nil {
			return nil, resolveToken, err
		}
		jobMutators = append(jobMutators, jobMutator)
	}
	return jobMutators, resolveToken, nil
}

func createMutator(mutatorConfig config.Mutator, loggerFactory *logutil.LoggerFactory, opaSDK *sdk.OPA) (admissionctrl.JobMutator, error) {
	switch mutatorConfig.Type {
	case "opa_json_patch":
		if mutatorConfig.OpaRule == nil {
			return nil, fmt.Errorf("mutator %q requires an opa_rule block", mutatorConfig.Name)
		}
		notationVerifier, err := buildVerifierIfEnabled(mutatorConfig.OpaRule.Notation, loggerFactory.GetLogger("notation_verifier"))
		if err != nil {
			return nil, err
		}
		return mutator.NewOpaJsonPatchMutator(mutatorConfig.Name, mutatorConfig.OpaRule.Filename, mutatorConfig.OpaRule.Query, loggerFactory.GetLogger("opa_mutator"), notationVerifier)
	case "json_patch_webhook":
		if mutatorConfig.Webhook == nil {
			return nil, fmt.Errorf("mutator %q requires a webhook block", mutatorConfig.Name)
		}
		return mutator.NewJsonPatchWebhookMutator(mutatorConfig.Name, mutatorConfig.Webhook.Endpoint, mutatorConfig.Webhook.Method, loggerFactory.GetLogger("json_patch_webhook_mutator"))
	case "opa_bundle_json_patch":
		if mutatorConfig.OpaSdkRule == nil {
			return nil, fmt.Errorf("mutator %q requires an opa_sdk_rule block", mutatorConfig.Name)
		}
		return mutator.NewOpaBundleMutator(mutatorConfig.Name, mutatorConfig.OpaSdkRule.Path, loggerFactory.GetLogger("opa_bundle_mutator"), opaSDK)
	default:
		return nil, fmt.Errorf("unknown mutator type %s", mutatorConfig.Type)
	}
}

func createValidators(c *config.Config, loggerFactory *logutil.LoggerFactory, opaSDK *sdk.OPA) ([]admissionctrl.JobValidator, bool, error) {
	jobValidators := make([]admissionctrl.JobValidator, 0, len(c.Validators))
	var resolveToken bool
	for _, validatorConfig := range c.Validators {
		resolveToken = resolveToken || validatorConfig.ResolveToken
		jobValidator, err := createValidator(validatorConfig, loggerFactory, opaSDK)
		if err != nil {
			return nil, resolveToken, err
		}
		jobValidators = append(jobValidators, jobValidator)
	}
	return jobValidators, resolveToken, nil
}

func createValidator(validatorConfig config.Validator, loggerFactory *logutil.LoggerFactory, opaSDK *sdk.OPA) (admissionctrl.JobValidator, error) {
	switch validatorConfig.Type {
	case "opa":
		if validatorConfig.OpaRule == nil {
			return nil, fmt.Errorf("validator %q requires an opa_rule block", validatorConfig.Name)
		}
		notationVerifier, err := buildVerifierIfEnabled(validatorConfig.Notation, loggerFactory.GetLogger("notation_verifier"))
		if err != nil {
			return nil, err
		}
		return validator.NewOpaValidator(validatorConfig.Name, validatorConfig.OpaRule.Filename, validatorConfig.OpaRule.Query, loggerFactory.GetLogger("opa_validator"), notationVerifier)
	case "opa_bundle":
		if validatorConfig.OpaSdkRule == nil {
			return nil, fmt.Errorf("validator %q requires an opa_sdk_rule block", validatorConfig.Name)
		}
		return validator.NewOpaBundleValidator(validatorConfig.Name, validatorConfig.OpaSdkRule.Path, loggerFactory.GetLogger("opa_bundle_validator"), opaSDK)
	case "webhook":
		if validatorConfig.Webhook == nil {
			return nil, fmt.Errorf("validator %q requires a webhook block", validatorConfig.Name)
		}
		return validator.NewWebhookValidator(validatorConfig.Name, validatorConfig.Webhook.Endpoint, validatorConfig.Webhook.Method, loggerFactory.GetLogger("webhook_validator"))
	case "notation":
		notationVerifier, err := buildVerifier(validatorConfig.Notation, loggerFactory.GetLogger("notation_verifier"))
		if err != nil {
			return nil, err
		}
		return validator.NewNotationValidator(loggerFactory.GetLogger("notation_validator"), validatorConfig.Name, notationVerifier), nil
	default:
		return nil, fmt.Errorf("unknown validator type %s", validatorConfig.Type)
	}
}
func buildVerifierIfEnabled(notationVerifierConfig *config.NotationVerifierConfig, logger *slog.Logger) (notation.ImageVerifier, error) {
	if notationVerifierConfig == nil {
		return nil, nil
	}
	return buildVerifier(notationVerifierConfig, logger)
}
func buildVerifier(notationVerifierConfig *config.NotationVerifierConfig, logger *slog.Logger) (notation.ImageVerifier, error) {

	if notationVerifierConfig == nil {
		return nil, fmt.Errorf("notation verifier config is nil")
	}
	policy, err := notation.LoadTrustPolicyDocument(notationVerifierConfig.TrustPolicyFile)
	if err != nil {
		return nil, err
	}
	ts := truststore.NewX509TrustStore(dir.NewSysFS(notationVerifierConfig.TrustStoreDir))

	return notation.NewImageVerifier(policy, ts, notationVerifierConfig.RepoPlainHTTP, notationVerifierConfig.MaxSigAttempts, notationVerifierConfig.CredentialStoreFile, logger)
}

func buildTlsConfig(config config.NomadServerTLS) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	if config.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if config.CaFile != "" {
		caCert, err := os.ReadFile(config.CaFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("CA file %q does not contain a valid certificate", config.CaFile)
		}
		tlsConfig.RootCAs = caCertPool
	}

	return tlsConfig, nil
}
func setupOpaSDK(ctx context.Context, logger *slog.Logger, opaConfig *config.OpaSdk) (*sdk.OPA, func(), error) {
	if opaConfig == nil {
		return nil, nil, nil
	}

	opaSDK, err := buildOpaSdk(ctx, logger, opaConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build OPA SDK: %w", err)
	}

	return opaSDK, func() { stopOpaSDK(opaSDK) }, nil
}

func buildOpaSdk(ctx context.Context, logger *slog.Logger, opaConfig *config.OpaSdk) (*sdk.OPA, error) {
	return buildOpaSdkWithTimeout(ctx, logger, opaConfig, 30*time.Second)
}

func buildOpaSdkWithTimeout(ctx context.Context, logger *slog.Logger, opaConfig *config.OpaSdk, readyTimeout time.Duration) (*sdk.OPA, error) {
	logger.Info("Starting OPA SDK", "config_path", opaConfig.ConfigPath, "id", opaConfig.Id)
	f, err := os.Open(opaConfig.ConfigPath)
	if err != nil {
		return nil, err
	}
	defer closeOpaConfig(f, logger)

	ready := make(chan struct{})
	readyCtx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()

	opaSDK, err := sdk.New(ctx, sdk.Options{
		ID:     opaConfig.Id,
		Config: f,
		Ready:  ready,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA SDK: %w", err)
	}

	logger.Info("Waiting for OPA SDK to become ready", "id", opaConfig.Id)
	select {
	case <-ready:
		logger.Info("OPA SDK is ready", "id", opaConfig.Id)
		return opaSDK, nil
	case <-readyCtx.Done():
		stopOpaSDK(opaSDK)
		logger.Error("OPA SDK did not become ready in time", "id", opaConfig.Id)
		return nil, fmt.Errorf("OPA SDK did not become ready in time: %w", readyCtx.Err())
	}
}

func stopOpaSDK(opaSDK *sdk.OPA) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	opaSDK.Stop(shutdownCtx)
}

func closeOpaConfig(file *os.File, logger *slog.Logger) {
	if err := file.Close(); err != nil {
		logger.Warn("Closing OPA SDK configuration failed", "error", err)
	}
}
