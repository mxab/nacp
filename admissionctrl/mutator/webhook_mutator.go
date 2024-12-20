package mutator

import (
	"bytes"
	"encoding/json"
	"github.com/mxab/nacp/admissionctrl/types"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/nomad/api"
)

type WebhookMutator struct {
	name     string
	endpoint *url.URL
	method   string
}

func (w *WebhookMutator) Mutate(payload *types.Payload) (out *api.Job, warnings []error, err error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequest(w.method, w.endpoint.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, nil, err
	}

	// Add context headers and body if available
	if payload.Context != nil {
		// Add standard headers for backward compatibility
		if payload.Context.ClientIP != "" {
			req.Header.Set("X-Forwarded-For", payload.Context.ClientIP) // Standard proxy header
			req.Header.Set("NACP-Client-IP", payload.Context.ClientIP)  // NACP specific
		}
		if payload.Context.AccessorID != "" {
			req.Header.Set("NACP-Accessor-ID", payload.Context.AccessorID)
		}
	}

	req.Body = io.NopCloser(bytes.NewBuffer(data))
	req.ContentLength = int64(len(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	newJob := &api.Job{}
	err = json.NewDecoder(resp.Body).Decode(newJob)
	if err != nil {
		return nil, nil, err
	}
	return newJob, nil, nil
}
func (w *WebhookMutator) Name() string {
	return w.name
}

func NewWebhookMutator(name string, endpoint *url.URL, method string) *WebhookMutator {

	return &WebhookMutator{
		name:     name,
		endpoint: endpoint,
		method:   method,
	}
}
