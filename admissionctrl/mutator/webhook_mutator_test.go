package mutator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

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
		job *api.Job
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantOut      *api.Job
		wantWarnings []error
		wantErr      bool
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
			gotOut, gotWarnings, err := w.Mutate(tt.args.job)
			assert.True(t, endpointCalled, "Ensure endpoint was called")
			if (err != nil) != tt.wantErr {
				t.Errorf("WebhookMutator.Mutate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, gotOut, tt.wantOut, "Ensure job is set")
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
