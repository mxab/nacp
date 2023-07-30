package main

import (
	"compress/gzip"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-multierror"

	"github.com/hashicorp/nomad/api"
	"github.com/hashicorp/nomad/helper"
	"github.com/hashicorp/nomad/helper/tlsutil"
	"github.com/hashicorp/nomad/lib/file"
	"github.com/mxab/nacp/admissionctrl"
	"github.com/mxab/nacp/admissionctrl/mutator"
	"github.com/mxab/nacp/admissionctrl/validator"
	"github.com/mxab/nacp/config"
	"github.com/mxab/nacp/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// rewrite the test above as table driven test
func TestProxy(t *testing.T) {

	type test struct {
		name   string
		path   string
		method string

		requestSender         func(*api.Client) (interface{}, *api.WriteMeta, error)
		wantNomadRequestJson  string
		wantProxyResponse     interface{}
		nomadResponse         string
		nomadResponseEncoding string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator
	}

	tests := []test{

		{
			name:   "create job adds hello meta",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},
			wantNomadRequestJson: registerRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponse: &api.JobRegisterResponse{},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&testutil.HelloMutator{},
			},
		},

		{
			name:   "plan job adds hello meta",
			path:   "/v1/job/example/plan",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, jobWithHelloWorldMeta(t)),

			wantProxyResponse: &api.JobPlanResponse{},

			nomadResponse: toJson(t, &api.JobPlanResponse{}),

			validators: []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&testutil.HelloMutator{},
			},
		},
		{
			name:   "plan job appends warning",
			path:   "/v1/job/example/plan",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobPlanResponse{
				// TODO: rework error concatination
				Warnings: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobPlanResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),

			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "plan job appends warning with gzip encoding",
			path:   "/v1/job/example/plan",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobPlanResponse{
				// TODO: rework error concatination
				Warnings: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobPlanResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),
			nomadResponseEncoding: "gzip",

			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "create job adds warnings",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobRegisterResponse{
				Warnings: "1 warning:\n\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "create job concats warnings",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobRegisterResponse{
				// TODO: rework error concatination
				Warnings: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "create job concats warnings if encoding is gzip",
			path:   "/v1/jobs",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Register(testutil.ReadJob(t, "job.json"), nil)
			},

			wantNomadRequestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobRegisterResponse{
				// TODO: rework error concatination
				Warnings: "2 warnings:\n\n* 1 error occurred:\n\t* some warning\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobRegisterResponse{
				Warnings: multierror.Append(nil, fmt.Errorf("some warning")).Error(),
			}),
			nomadResponseEncoding: "gzip",
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "plan job adds warnings",
			path:   "/v1/job/example/plan",
			method: "PUT",
			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Plan(testutil.ReadJob(t, "job.json"), false, nil)
			},

			wantNomadRequestJson: planRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantProxyResponse: &api.JobPlanResponse{
				Warnings: "1 warning:\n\n* some warning",
			},

			nomadResponse: toJson(t, &api.JobPlanResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "validate job adds hello meta",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(testutil.ReadJob(t, "job.json"), nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: jobWithHelloWorldMeta(t)}),

			wantProxyResponse: &api.JobValidateResponse{},

			nomadResponse: toJson(t, &api.JobValidateResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&testutil.HelloMutator{},
			},
		},
		{
			name:   "validate job appends warnings",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(&api.Job{}, nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),

			wantProxyResponse: &api.JobValidateResponse{
				Warnings: helper.MergeMultierrorWarnings(errors.New("some warning")),
			},

			nomadResponse: toJson(t, &api.JobValidateResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "validate job appends validation errors",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(&api.Job{}, nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),

			wantProxyResponse: &api.JobValidateResponse{
				ValidationErrors: []string{"some error"},
				Error:            "1 error occurred:\n\t* some error\n\n",
			},

			nomadResponse: toJson(t, &api.JobValidateResponse{}),
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningError("some error"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
		{
			name:   "validate job appends warnings and handles gzip",
			path:   "/v1/validate/job",
			method: "PUT",

			requestSender: func(c *api.Client) (interface{}, *api.WriteMeta, error) {
				return c.Jobs().Validate(&api.Job{}, nil)
			},
			wantNomadRequestJson: toJson(t, &api.JobValidateRequest{Job: &api.Job{}}),

			wantProxyResponse: &api.JobValidateResponse{
				Warnings: helper.MergeMultierrorWarnings(errors.New("some warning")),
			},

			nomadResponse:         toJson(t, &api.JobValidateResponse{}),
			nomadResponseEncoding: "gzip",
			validators: []admissionctrl.JobValidator{
				mockValidatorReturningWarnings("some warning"),
			},
			mutators: []admissionctrl.JobMutator{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadBackendCalled := false
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				nomadBackendCalled = true
				assert.Equal(t, req.Method, tc.method, "Ensure method is set")
				assert.Equal(t, req.URL.Path, tc.path, "Ensure path is set")
				jsonData, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				json := string(jsonData)
				assert.JSONEq(t, tc.wantNomadRequestJson, json, "Body matches")

				//set encoding to gzip
				if tc.nomadResponseEncoding == "gzip" {
					rw.Header().Set("Content-Encoding", "gzip")
					rw.WriteHeader(http.StatusOK)
					//write gzip response
					gzipWriter := gzip.NewWriter(rw)
					defer gzipWriter.Close()
					gzipWriter.Write([]byte(tc.nomadResponse))

				} else {
					rw.WriteHeader(http.StatusOK)
					rw.Write([]byte(tc.nomadResponse))
				}

			}))
			// Close the server when test finishes
			defer nomadDummy.Close()

			// Use Client & URL from our local test server

			nomad, err := url.Parse(nomadDummy.URL)
			if err != nil {
				t.Fatal(err)
			}
			jobHandler := admissionctrl.NewJobHandler(
				tc.mutators,
				tc.validators,
				hclog.NewNullLogger(),
			)
			proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger(), nil)

			proxyServer := httptest.NewServer(http.HandlerFunc(proxy))
			defer proxyServer.Close()
			nomadClient := buildNomadClient(t, proxyServer)

			resp, _, err := tc.requestSender(nomadClient)

			assert.NoError(t, err, "No http call error")
			assert.Equal(t, tc.wantProxyResponse, resp, "OK response is expected")

			assert.True(t, nomadBackendCalled, "Nomad backend was called")

		})
	}

}
func TestJobUpdateProxy(t *testing.T) {

	type test struct {
		name        string
		path        string
		method      string
		requestJson string

		wantNomadRequestJson  string
		wantProxyResponseJson string
		nomadResponse         string
		//	responseWarnings []error
		validators []admissionctrl.JobValidator
		mutators   []admissionctrl.JobMutator
	}

	tests := []test{

		{
			name:        "update job adds hello meta (does this method actually exist?)",
			path:        "/v1/job/example",
			method:      "PUT",
			requestJson: registerRequestJson(t, testutil.ReadJob(t, "job.json")),

			wantNomadRequestJson:  registerRequestJson(t, jobWithHelloWorldMeta(t)),
			wantProxyResponseJson: toJson(t, &api.JobRegisterResponse{}),

			nomadResponse: toJson(t, &api.JobRegisterResponse{}),
			validators:    []admissionctrl.JobValidator{},
			mutators: []admissionctrl.JobMutator{
				&testutil.HelloMutator{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			nomadBackendCalled := false
			nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				nomadBackendCalled = true
				assert.Equal(t, req.Method, tc.method, "Ensure method is set")
				assert.Equal(t, req.URL.Path, tc.path, "Ensure path is set")
				jsonData, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatal(err)
				}
				json := string(jsonData)
				assert.JSONEq(t, tc.wantNomadRequestJson, json, "Body matches")

				_, _ = rw.Write([]byte(tc.nomadResponse))
			}))
			// Close the server when test finishes
			defer nomadDummy.Close()

			// Use Client & URL from our local test server

			nomad, err := url.Parse(nomadDummy.URL)
			if err != nil {
				t.Fatal(err)
			}
			jobHandler := admissionctrl.NewJobHandler(
				tc.mutators,
				tc.validators,
				hclog.NewNullLogger(),
			)
			proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger(), nil)

			proxyServer := httptest.NewServer(http.HandlerFunc(proxy))
			defer proxyServer.Close()

			res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, tc.path), strings.NewReader(tc.requestJson))
			assert.NoError(t, err, "No http call error")
			assert.Equal(t, 200, res.StatusCode, "OK response is expected")
			assert.JSONEq(t, tc.wantProxyResponseJson, readClosterToString(t, res.Body), "Body matches")
			assert.True(t, nomadBackendCalled, "Nomad backend was called")

		})
	}

}

func buildNomadClient(t *testing.T, proxyServer *httptest.Server) *api.Client {
	t.Helper()
	nomadClient, err := api.NewClient(&api.Config{
		Address: proxyServer.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	return nomadClient
}
func mockValidatorReturningWarnings(warning string) admissionctrl.JobValidator {

	validator := new(testutil.MockValidator)
	validator.On("Validate", mock.Anything).Return([]error{fmt.Errorf(warning)}, nil)
	return validator
}
func mockValidatorReturningError(err string) admissionctrl.JobValidator {

	validator := new(testutil.MockValidator)
	validator.On("Validate", mock.Anything).Return([]error{}, fmt.Errorf(err))
	return validator
}

func readClosterToString(t *testing.T, rc io.ReadCloser) string {
	t.Helper()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func jobWithHelloWorldMeta(t *testing.T) *api.Job {
	wantJob := testutil.ReadJob(t, "job.json")
	wantJob.Meta = map[string]string{
		"hello": "world",
	}
	return wantJob
}
func toJson(t *testing.T, v interface{}) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func registerRequestJson(t *testing.T, wantJob *api.Job) string {
	t.Helper()
	register := &api.JobRegisterRequest{
		Job: wantJob,
	}
	return toJson(t, register)

}

func planRequestJson(t *testing.T, wantJob *api.Job) string {
	t.Helper()
	plan := &api.JobPlanRequest{
		Job: wantJob,
	}
	return toJson(t, plan)

}

func TestAdmissionControllerErrors(t *testing.T) {
	nomadDummy := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {

		_, _ = rw.Write([]byte(`you should not see this`))
	}))
	// Close the server when test finishes
	defer nomadDummy.Close()

	validator := new(testutil.MockValidator)
	validator.On("Validate", mock.Anything).Return([]error{}, fmt.Errorf("some error"))

	nomad, err := url.Parse(nomadDummy.URL)
	if err != nil {
		t.Fatal(err)
	}
	jobHandler := admissionctrl.NewJobHandler(
		[]admissionctrl.JobMutator{},
		[]admissionctrl.JobValidator{validator},
		hclog.NewNullLogger(),
	)
	proxy := NewProxyHandler(nomad, jobHandler, hclog.NewNullLogger(), nil)

	proxyServer := httptest.NewServer(http.HandlerFunc(proxy))

	defer proxyServer.Close()

	jobRequestJson := registerRequestJson(t, testutil.ReadJob(t, "job.json"))
	res, err := sendPut(t, fmt.Sprintf("%s%s", proxyServer.URL, "/v1/jobs"), strings.NewReader(jobRequestJson))
	require.NoError(t, err, "No http call error")
	assert.Equal(t, 500, res.StatusCode, "Should return 400")
	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)

}

func sendPut(t *testing.T, url string, body io.Reader) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func TestDefaultBuildServer(t *testing.T) {
	logger := hclog.NewNullLogger()
	c := buildConfig(logger)
	server, err := buildServer(c, logger)
	assert.NoError(t, err)

	assert.NotNil(t, server)

}

func TestCreateValidators(t *testing.T) {

	tt := []struct {
		name       string
		validators config.Validator
		want       admissionctrl.JobValidator
		wantErr    bool
	}{

		{
			name: "opa validator",
			validators: config.Validator{

				Type: "opa",
				Name: "test",
				OpaRule: &config.OpaRule{
					Query:    "errors = data.dummy.errors",
					Filename: testutil.Filepath(t, "opa/errors.rego"),
				},
			},
			want: &validator.OpaValidator{},
		},
		{
			name: "webhook validator",
			validators: config.Validator{

				Type: "webhook",
				Name: "test",
				Webhook: &config.Webhook{
					Endpoint: "http://example.com",
					Method:   "PUT",
				},
			},
			want: &validator.WebhookValidator{},
		},
		{
			name: "invalid validator type",
			validators: config.Validator{

				Type: "invalid",
				Name: "test",
				OpaRule: &config.OpaRule{
					Query:    "errors = data.dummy.errors",
					Filename: testutil.Filepath(t, "opa/errors.rego"),
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := &config.Config{
				Validators: []config.Validator{tc.validators},
			}

			validators, err := createValidators(c, hclog.NewNullLogger())

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.IsType(t, tc.want, validators[0])

		})

	}
}

func TestCreateMutatators(t *testing.T) {
	tt := []struct {
		name     string
		mutators config.Mutator
		want     admissionctrl.JobMutator
		wantErr  bool
	}{
		{
			name: "opa json patch mutator",
			mutators: config.Mutator{

				Type: "opa_json_patch",
				Name: "test",
				OpaRule: &config.OpaRule{
					Query:    "errors = data.dummy.errors",
					Filename: testutil.Filepath(t, "opa/errors.rego"),
				},
			},
			want: &mutator.OpaJsonPatchMutator{},
		},
		{
			name: "webhook json patch mutator",
			mutators: config.Mutator{

				Type: "json_patch_webhook",
				Name: "test",
				Webhook: &config.Webhook{
					Endpoint: "http://example.com",
					Method:   "PUT",
				},
			},
			want: &mutator.JsonPatchWebhookMutator{},
		},
		{
			name: "invalid mutator type",
			mutators: config.Mutator{

				Type: "invalid",
				Name: "test",
				OpaRule: &config.OpaRule{
					Query:    "errors = data.dummy.errors",
					Filename: testutil.Filepath(t, "opa/errors.rego"),
				},
			},
			wantErr: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			c := &config.Config{
				Mutators: []config.Mutator{tc.mutators},
			}

			mutators, err := createMutators(c, hclog.NewNullLogger())

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			assert.IsType(t, tc.want, mutators[0])

		})

	}
}

func TestCreateTlsConfig(t *testing.T) {
	caCertFileName, _, _, _, cleanup := generateTLSData(t)
	defer cleanup()
	config, err := createTlsConfig(caCertFileName)
	assert.NoError(t, err)
	assert.NotNil(t, config)
}
func TestBuildCustomTransport(t *testing.T) {

	caCertFileName, _, certFileName, pkFileName, cleanup := generateTLSData(t)
	defer cleanup()

	tls := config.NomadServerTLS{
		CaFile:   caCertFileName,
		CertFile: certFileName,
		KeyFile:  pkFileName,
	}
	transport, err := buildCustomTransport(tls)
	assert.NoError(t, err)
	assert.NotNil(t, transport)

}

func generateTLSData(t *testing.T) (caCertFileName, caPkFileName, certFileName, pkFileName string, cleanup func()) {
	t.Helper()

	dir := t.TempDir()
	cleanup = func() {
		os.RemoveAll(dir)
	}
	domain := "nomad"
	days := 1

	caCertFileName = fmt.Sprintf("%s/%s-agent-ca.pem", dir, domain)
	caPkFileName = fmt.Sprintf("%s/%s-agent-ca-key.pem", dir, domain)

	//	constraints := []string{}
	constraints := []string{domain, "localhost"}
	commonName := ""

	ca, caPk, err := tlsutil.GenerateCA(tlsutil.CAOpts{Name: commonName, Days: days, PermittedDNSDomains: constraints})

	if err != nil {
		t.Fatal(err)
	}

	writeTLSStuff(t, caCertFileName, ca)
	writeTLSStuff(t, caPkFileName, caPk)

	cluster_region := "global"

	var DNSNames []string
	var IPAddresses []net.IP
	var extKeyUsage []x509.ExtKeyUsage
	var name, prefix string

	server := true
	client := false
	if server {
		name = fmt.Sprintf("server.%s.%s", cluster_region, domain)
		DNSNames = append(DNSNames, name)
		DNSNames = append(DNSNames, "localhost")

		IPAddresses = append(IPAddresses, net.ParseIP("127.0.0.1"))
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
		prefix = fmt.Sprintf("%s-server-%s", cluster_region, domain)

	} else if client {
		name = fmt.Sprintf("client.%s.%s", cluster_region, domain)
		DNSNames = append(DNSNames, []string{name, "localhost"}...)
		IPAddresses = append(IPAddresses, net.ParseIP("127.0.0.1"))
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth}
		prefix = fmt.Sprintf("%s-client-%s", cluster_region, domain)
	}

	certFileName = fmt.Sprintf("%s/%s.pem", dir, prefix)
	pkFileName = fmt.Sprintf("%s/%s-key.pem", dir, prefix)

	signer, err := tlsutil.ParseSigner(string(caPk))
	if err != nil {
		t.Fatal(err)
	}

	pub, priv, err := tlsutil.GenerateCert(tlsutil.CertOpts{
		Signer: signer, CA: ca, Name: name, Days: days,
		DNSNames: DNSNames, IPAddresses: IPAddresses, ExtKeyUsage: extKeyUsage,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = tlsutil.Verify(ca, pub, name); err != nil {
		t.Fatal(err)
	}

	writeTLSStuff(t, certFileName, pub)

	writeTLSStuff(t, pkFileName, priv)
	return

}
func writeTLSStuff(t *testing.T, name, data string) {
	t.Helper()
	if err := file.WriteAtomicWithPerms(name, []byte(data), 0755, 0600); err != nil {
		t.Fatal(err)
	}
}
