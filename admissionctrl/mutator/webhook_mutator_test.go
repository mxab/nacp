package mutator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/mxab/nacp/admissionctrl/types"
	"github.com/mxab/nacp/config"

	"github.com/hashicorp/nomad/api"
	"github.com/stretchr/testify/assert"
)

func TestWebhookMutator_Mutate(t *testing.T) {
	type fields struct {
		name         string
		endpointPath string
		method       string
	}
	type args struct {
		job     *api.Job
		context *config.RequestContext
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantOut      *api.Job
		wantMutated  bool
		wantWarnings []error
		wantErr      bool

		wantedHeaders map[string]string
	}{
		{
			name: "Test simple mutation",
			fields: fields{
				name:         "test",
				endpointPath: "/test",
				method:       "POST",
			},
			args: args{
				job: &api.Job{},
			},
			wantOut: &api.Job{
				Meta: map[string]string{
					"test": "test",
				},
			},
			wantMutated: true,
		},
		{
			name: "with context client ip",
			fields: fields{
				name:         "test",
				endpointPath: "/test",
				method:       "POST",
			},
			args: args{
				job: &api.Job{},
				context: &config.RequestContext{
					ClientIP: "127.0.0.1",
				},
			},
			wantOut: &api.Job{
				Meta: map[string]string{
					"test": "test",
				},
			},
			wantMutated: true,
			wantedHeaders: map[string]string{
				"X-Forwarded-For": "127.0.0.1",
				"NACP-Client-IP":  "127.0.0.1",
			},
		},
		{
			name: "with context accessor id",
			fields: fields{
				name:         "test",
				endpointPath: "/test",
				method:       "POST",
			},
			args: args{
				job: &api.Job{},
				context: &config.RequestContext{
					AccessorID: "1234",
				},
			},
			wantOut: &api.Job{
				Meta: map[string]string{
					"test": "test",
				},
			},
			wantMutated: true,
			wantedHeaders: map[string]string{
				"NACP-Accessor-ID": "1234",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			endpointCalled := false
			testEndpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				endpointCalled = true
				assert.Equal(t, req.Method, tt.fields.method, "Ensure method is set")
				assert.Equal(t, req.URL.Path, tt.fields.endpointPath, "Ensure path is set")
				assert.Equal(t, req.Header.Get("Content-Type"), "application/json", "Ensure content type is set")
				job := &api.Job{}
				json.NewDecoder(req.Body).Decode(job)
				assert.Equal(t, job, tt.args.job, "Ensure job is set")
				for key, value := range tt.wantedHeaders {
					assert.Equal(t, value, req.Header.Get(key), "Header %s does not match", key)
				}
				rw.WriteHeader(http.StatusOK)
				json.NewEncoder(rw).Encode(tt.wantOut)

			}))
			defer testEndpoint.Close()

			endpoint, _ := url.Parse(fmt.Sprintf("%s%s", testEndpoint.URL, tt.fields.endpointPath))

			w := &WebhookMutator{
				name:     tt.fields.name,
				endpoint: endpoint,
				method:   tt.fields.method,
			}
			payload := &types.Payload{Job: tt.args.job, Context: tt.args.context}
			gotOut, gotMutated, gotWarnings, err := w.Mutate(t.Context(), payload)
			assert.True(t, endpointCalled, "Ensure endpoint was called")
			if (err != nil) != tt.wantErr {
				t.Errorf("WebhookMutator.Mutate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, gotOut, tt.wantOut, "Ensure job is set")
			assert.Equal(t, tt.wantMutated, gotMutated, "WebhookMutator.Mutate() mutated = %v, want %v", gotMutated, tt.wantMutated)
			assert.Equal(t, gotWarnings, tt.wantWarnings, "Ensure warnings are set")

		})
	}
}

func TestNewWebhookMutator(t *testing.T) {
	type args struct {
		name     string
		endpoint *url.URL
		method   string
	}
	tests := []struct {
		name string
		args args
		want *WebhookMutator
	}{
		{
			name: "test",
			args: args{
				name:     "test",
				endpoint: mustParse(t, "http://localhost:8080/foo/bar"),
				method:   "POST",
			},
			want: &WebhookMutator{
				name:     "test",
				endpoint: mustParse(t, "http://localhost:8080/foo/bar"),
				method:   "POST",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewWebhookMutator(tt.args.name, tt.args.endpoint, tt.args.method); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewWebhookMutator() = %v, want %v", got, tt.want)
				assert.Equal(t, got.Name(), tt.args.name, "Name is not equal")
			}
		})
	}
}
func mustParse(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}
